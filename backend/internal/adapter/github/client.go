package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/shurcooL/graphql"
)

type Client struct {
	graphqlClient *graphql.Client
	perPage       int
	log           *slog.Logger
}

const (
	minContributors  = 10
	maxLastCommitAge = 90 * 24 * time.Hour // 90 days
	maxComments      = 50
)

type authTransport struct {
	token string
	rt    http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.rt.RoundTrip(req)
}

// NewClient creates a GitHub GraphQL client.
func NewClient(token string, perPage int, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	httpClient := &http.Client{
		Transport: &authTransport{
			token: token,
			rt:    http.DefaultTransport,
		},
		Timeout: 20 * time.Second,
	}
	return &Client{
		graphqlClient: graphql.NewClient("https://api.github.com/graphql", httpClient),
		perPage:       perPage,
		log:           log,
	}
}

// FetchIssues fetches issues for each language/label pair and returns typed domain issues.
func (c *Client) FetchIssues(ctx context.Context, languages, labels []string) ([]domain.Issue, error) {
	result := make([]domain.Issue, 0)
	seen := make(map[string]struct{})

	for _, lang := range languages {
		for _, label := range labels {
			// Push recency and comment-count caps into the search query to
			// reduce the result set returned by GitHub.
			createdCutoff := time.Now().Add(-maxLastCommitAge).Format("2006-01-02")
			queryStr := fmt.Sprintf(`label:"%s" language:%s state:open sort:created-desc created:>=%s comments:<=%d`, label, lang, createdCutoff, maxComments)

			var q struct {
				Search struct {
					Nodes []struct {
						ID        graphql.ID
						Title     graphql.String
						URL       graphql.String
						Body      *graphql.String
						CreatedAt graphql.String `graphql:"createdAt"`
						Comments  struct {
							TotalCount graphql.Int `graphql:"totalCount"`
						} `graphql:"comments"`
						Labels struct {
							Nodes []struct {
								Name graphql.String
							} `graphql:"nodes"`
						} `graphql:"labels(first:10)"`
						Repository struct {
							NameWithOwner  graphql.String `graphql:"nameWithOwner"`
							Description    *graphql.String
							StargazerCount graphql.Int `graphql:"stargazerCount"`
							LicenseInfo    *struct {
								Name *graphql.String
							} `graphql:"licenseInfo"`
							MentionableUsers *struct {
								TotalCount graphql.Int `graphql:"totalCount"`
							} `graphql:"mentionableUsers(first:1)"`
							Readme *struct {
								ID graphql.ID
							} `graphql:"object(expression: \"HEAD:README.md\")"`
							Contributing *struct {
								ID graphql.ID
							} `graphql:"object(expression: \"HEAD:CONTRIBUTING.md\")"`
							DefaultBranchRef *struct {
								Target *struct {
									History struct {
										Nodes []struct {
											CommittedDate graphql.String `graphql:"committedDate"`
										} `graphql:"nodes"`
									} `graphql:"history(first:1)"`
								} `graphql:"target"`
							} `graphql:"defaultBranchRef"`
						} `graphql:"repository"`
					} `graphql:"nodes"`
				} `graphql:"search(query: $query, type: ISSUE, first: $perPage)"`
			}

			variables := map[string]interface{}{
				"query":   graphql.String(queryStr),
				"perPage": graphql.Int(c.perPage),
			}

			if err := c.graphqlClient.Query(ctx, &q, variables); err != nil {
				c.log.Error("github query failed", "language", lang, "label", label, "error", err)
				continue
			}

			for _, node := range q.Search.Nodes {
				id := fmt.Sprint(node.ID)
				if id == "" {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}

				createdAt := parseTime(string(node.CreatedAt))

				labelsOut := make([]string, 0, len(node.Labels.Nodes))
				for _, ln := range node.Labels.Nodes {
					labelsOut = append(labelsOut, string(ln.Name))
				}
				labelsJSON, err := json.Marshal(labelsOut)
				if err != nil {
					labelsJSON = []byte("[]")
				}

				contributorCount := 0
				if node.Repository.MentionableUsers != nil {
					contributorCount = int(node.Repository.MentionableUsers.TotalCount)
				}

				var lastCommit *time.Time
				if node.Repository.DefaultBranchRef != nil && node.Repository.DefaultBranchRef.Target != nil {
					h := node.Repository.DefaultBranchRef.Target.History
					if len(h.Nodes) > 0 {
						if t := parseTime(string(h.Nodes[0].CommittedDate)); !t.IsZero() {
							lastCommit = &t
						}
					}
				}

				repo := domain.Repository{
					Name:             string(node.Repository.NameWithOwner),
					Description:      ptrToString(node.Repository.Description),
					Stars:            int(node.Repository.StargazerCount),
					LicenseName:      licenseName(node.Repository.LicenseInfo),
					ContributorCount: contributorCount,
					HasReadme:        node.Repository.Readme != nil || node.Repository.Contributing != nil,
					HasContributing:  node.Repository.Contributing != nil,
					LastCommitAt:     lastCommit,
				}

				bodyText := ""
				if node.Body != nil {
					bodyText = string(*node.Body)
				}

				issue := domain.Issue{
					ID:           id,
					Title:        string(node.Title),
					URL:          string(node.URL),
					Body:         bodyText,
					CreatedAt:    createdAt,
					CommentCount: int(node.Comments.TotalCount),
					Labels:       string(labelsJSON),
					Repository:   repo,
				}

				// Apply repo quality filters per README: require README, license,
				// minimum contributors and recent activity (last commit within 90 days).
				if repo.LicenseName == "" {
					continue
				}
				if !repo.HasReadme {
					continue
				}
				if repo.ContributorCount < minContributors {
					continue
				}
				if repo.LastCommitAt == nil || time.Since(*repo.LastCommitAt) > maxLastCommitAge {
					continue
				}

				result = append(result, issue)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("github query failed for all language/label combinations")
	}

	return result, nil
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func ptrToString(p *graphql.String) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

func licenseName(li *struct{ Name *graphql.String }) string {
	if li == nil || li.Name == nil {
		return ""
	}
	return string(*li.Name)
}

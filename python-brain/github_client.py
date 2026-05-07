"""
github_client.py

GitHub GraphQL API client.
"""

import logging
from datetime import datetime, timezone, timedelta
from typing import Optional

import httpx

from config import GitHubConfig, QualityFilter
from models import Issue, Repository

log = logging.getLogger(__name__)

_GRAPHQL_QUERY_TEMPLATE = """
query SearchIssues($query: String!) {
    search(query: $query, type: ISSUE, first: {per_page}) {
    nodes {
      ... on Issue {
        id
        title
        url
        body
        createdAt
        comments { totalCount }
        labels(first: 5) {
          nodes { name }
        }
        repository {
          nameWithOwner
          description
          stargazerCount
          licenseInfo { name }
          mentionableUsers(first: 1) { totalCount }
          hasReadme: object(expression: "HEAD:README.md") {
            ... on Blob { id }
          }
          hasContributing: object(expression: "HEAD:CONTRIBUTING.md") {
            ... on Blob { id }
          }
          defaultBranchRef {
            target {
              ... on Commit {
                history(first: 1) {
                  nodes { committedDate }
                }
              }
            }
          }
        }
      }
    }
  }
}
"""


class GitHubClient:
    def __init__(
        self,
        github_config: GitHubConfig,
        quality_filter: QualityFilter,
        token: str,
        http_client: httpx.Client | None = None,
    ) -> None:
        if not token:
            raise ValueError("GITHUB_TOKEN is required")
        self._config = github_config
        self._quality = quality_filter
        self._token = token
        self._client = http_client or httpx.Client(timeout=github_config.request_timeout_seconds)
        self._owns_client = http_client is None

    def __enter__(self) -> "GitHubClient":
        return self

    def __exit__(self, exc_type, exc, exc_tb) -> None:
        self.close()

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def fetch_all_issues(self) -> list[Issue]:
        """Fetch issues for all language/label combinations and apply filters."""
        seen_ids: set[str] = set()
        result: list[Issue] = []

        for language in self._config.languages:
            for label in self._config.labels:
                log.info("Fetching: %s / %s", language, label)
                issues = self._fetch_issues_for(language, label)

                for issue in issues:
                    if issue.id in seen_ids:
                        continue
                    seen_ids.add(issue.id)

                    if self._passes_quality_filter(issue.repository) and self._passes_issue_filter(issue):
                        result.append(issue)

        log.info("Qualified issues: %d", len(result))
        return result

    def _headers(self) -> dict:
        return {
            "Authorization": f"Bearer {self._token}",
            "Content-Type": "application/json",
        }

    def _build_search_query(self, language: str, label: str) -> str:
        return f'label:"{label}" language:{language} state:open sort:created-desc'

    def _parse_last_commit(self, repo_node: dict) -> Optional[datetime]:
        try:
            committed = (
                repo_node["defaultBranchRef"]["target"]["history"]["nodes"][0]["committedDate"]
            )
            return datetime.fromisoformat(committed.replace("Z", "+00:00"))
        except (KeyError, IndexError, TypeError):
            return None

    def _parse_repository(self, repo_node: dict) -> Repository:
        return Repository(
            name=repo_node.get("nameWithOwner", ""),
            description=repo_node.get("description") or "",
            stars=repo_node.get("stargazerCount", 0),
            license_name=(repo_node.get("licenseInfo") or {}).get("name", ""),
            contributor_count=repo_node.get("mentionableUsers", {}).get("totalCount", 0),
            has_readme=bool(repo_node.get("hasReadme")),
            has_contributing=bool(repo_node.get("hasContributing")),
            last_commit_at=self._parse_last_commit(repo_node),
        )

    def _parse_issue(self, node: dict) -> Optional[Issue]:
        """Parse a GraphQL node into an Issue model."""
        try:
            created_at = datetime.fromisoformat(
                node["createdAt"].replace("Z", "+00:00")
            )
        except (KeyError, ValueError):
            return None

        issue_id = node.get("id")
        title = node.get("title")
        url = node.get("url")
        if not issue_id or not title or not url:
            return None

        body = (node.get("body") or "")[: self._config.issue_body_max_length]
        labels = [label["name"] for label in node.get("labels", {}).get("nodes", [])]
        repository = self._parse_repository(node.get("repository", {}))

        return Issue(
            id=issue_id,
            title=title,
            url=url,
            body=body,
            created_at=created_at,
            comment_count=node.get("comments", {}).get("totalCount", 0),
            labels=labels,
            repository=repository,
        )

    def _passes_quality_filter(self, repo: Repository) -> bool:
        if not repo.has_readme:
            return False
        if not repo.license_name:
            return False
        if repo.contributor_count < self._quality.min_contributors:
            return False
        if self._quality.require_contributing and not repo.has_contributing:
            return False
        if repo.last_commit_at is None:
            return False

        cutoff = datetime.now(timezone.utc) - timedelta(days=self._quality.max_days_since_commit)
        return repo.last_commit_at >= cutoff

    def _passes_issue_filter(self, issue: Issue) -> bool:
        return issue.comment_count <= self._quality.max_comment_count

    def _fetch_issues_for(self, language: str, label: str) -> list[Issue]:
        variables = {"query": self._build_search_query(language, label)}
        # Use a simple replace to avoid interpreting other braces in the GraphQL
        # template as format placeholders.
        query = _GRAPHQL_QUERY_TEMPLATE.replace("{per_page}", str(self._config.issues_per_query))

        try:
            response = self._client.post(
                self._config.api_url,
                headers=self._headers(),
                json={"query": query, "variables": variables},
            )
            response.raise_for_status()
            data = response.json()

            if "errors" in data:
                log.warning(
                    "GraphQL error (%s/%s): %s",
                    language, label, data["errors"][0].get("message"),
                )
                return []

            nodes = data.get("data", {}).get("search", {}).get("nodes", [])
            return [issue for node in nodes if (issue := self._parse_issue(node))]

        except httpx.HTTPStatusError as e:
            log.error("HTTP %s (%s/%s)", e.response.status_code, language, label)
            return []
        except httpx.RequestError as e:
            log.error("Connection error (%s/%s): %s", language, label, e)
            return []
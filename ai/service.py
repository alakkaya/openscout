from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
from dotenv import load_dotenv
from datetime import datetime

load_dotenv()

# reuse project analyzer & models
from analyzer import IssueAnalyzer
from config import config as app_config, load_env_config
from models import Issue, Repository

from google import genai
from google.genai.errors import APIError

app = FastAPI()

class RepoPayload(BaseModel):
    name: str
    description: Optional[str] = ""
    stars: int = 0
    license_name: Optional[str] = ""
    contributor_count: Optional[int] = 0
    has_readme: Optional[bool] = False
    has_contributing: Optional[bool] = False
    last_commit_at: Optional[str] = None

class IssuePayload(BaseModel):
    id: str
    title: str
    url: str
    body: Optional[str] = ""
    created_at: str
    comment_count: int
    labels: List[str] = []
    repository: RepoPayload

class AnalyzeRequest(BaseModel):
    issues: List[IssuePayload]

class AnalysisOut(BaseModel):
    issue_id: str
    complexity: int
    estimated_hours: int
    skills_needed: List[str]
    why_solvable: str
    warning: Optional[str] = None

class AnalyzeResponse(BaseModel):
    analyses: List[AnalysisOut]

# build analyzer instance once
_env = load_env_config()
if not _env.gemini_api_key:
    raise RuntimeError("GEMINI_API_KEY not set")
_genai_client = genai.Client(api_key=_env.gemini_api_key)
_analyzer = IssueAnalyzer(app_config.analyzer, _genai_client)

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/analyze", response_model=AnalyzeResponse)
async def analyze(req: AnalyzeRequest):
    # convert payload -> domain.Issue list
    issues = []
    for p in req.issues:
        try:
            created_at = datetime.fromisoformat(p.created_at.replace("Z", "+00:00"))
        except (TypeError, ValueError):
            created_at = datetime.utcnow()
        repo = Repository(
            name=p.repository.name,
            description=p.repository.description or "",
            stars=p.repository.stars,
            license_name=p.repository.license_name or "",
            contributor_count=p.repository.contributor_count or 0,
            has_readme=bool(p.repository.has_readme),
            has_contributing=bool(p.repository.has_contributing),
            last_commit_at=(datetime.fromisoformat(p.repository.last_commit_at.replace("Z", "+00:00"))
                            if p.repository.last_commit_at else None),
        )
        issues.append(Issue(
            id=p.id,
            title=p.title,
            url=p.url,
            body=p.body or "",
            created_at=created_at,
            comment_count=p.comment_count,
            labels=p.labels,
            repository=repo,
        ))

    try:
        scored = _analyzer.analyze(issues)
    except APIError as e:
        raise HTTPException(status_code=502, detail=str(e)) from e
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e

    out = []
    for s in scored:
        a = s.analysis
        out.append(AnalysisOut(
            issue_id=s.issue.id,
            complexity=a.complexity,
            estimated_hours=a.estimated_hours,
            skills_needed=a.skills_needed,
            why_solvable=a.why_solvable,
            warning=a.warning,
        ))
    return AnalyzeResponse(analyses=out)
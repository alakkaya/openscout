"""
models.py

Uygulama genelinde kullanılan veri modelleri.
Dict yerine typed dataclass kullanmak: IDE desteği, hata yakalamayı
ve okunabilirliği artırır.
"""

from dataclasses import dataclass
from datetime import datetime
from typing import Optional


@dataclass
class Repository:
    name: str
    description: str
    stars: int
    license_name: str
    contributor_count: int
    has_readme: bool
    has_contributing: bool
    last_commit_at: Optional[datetime]


@dataclass
class Issue:
    id: str
    title: str
    url: str
    body: str
    created_at: datetime
    comment_count: int
    labels: list[str]
    repository: Repository


@dataclass
class IssueAnalysis:
    complexity: int
    estimated_hours: int
    skills_needed: list[str]
    why_solvable: str
    warning: Optional[str]


@dataclass
class ScoredIssue:
    issue: Issue
    analysis: IssueAnalysis

    @property
    def title(self) -> str:
        return self.issue.title

    @property
    def url(self) -> str:
        return self.issue.url

    @property
    def repo_name(self) -> str:
        return self.issue.repository.name

    @property
    def repo_stars(self) -> int:
        return self.issue.repository.stars
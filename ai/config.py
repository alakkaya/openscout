"""
config.py

Application configuration lives here.
"""

import os
from dataclasses import dataclass, field


@dataclass(frozen=True)
class GitHubConfig:
    api_url: str = "https://api.github.com/graphql"
    languages: tuple[str, ...] = ("Go", "Python", "TypeScript")
    labels: tuple[str, ...] = ("good first issue", "help wanted")
    issues_per_query: int = 20
    request_timeout_seconds: int = 15
    issue_body_max_length: int = 1000


@dataclass(frozen=True)
class QualityFilter:
    min_contributors: int = 10
    max_days_since_commit: int = 90
    max_comment_count: int = 15
    require_contributing: bool = False


@dataclass(frozen=True)
class AnalyzerConfig:
    model: str = "gemini-2.5-flash"
    max_tokens: int = 400
    max_complexity_score: int = 5
    top_issues_count: int = 5


@dataclass(frozen=True)
class SchedulerConfig:
    timezone: str = "Europe/Istanbul"
    hour: int = 8
    minute: int = 0
    misfire_grace_seconds: int = 300


@dataclass(frozen=True)
class AppConfig:
    github: GitHubConfig = field(default_factory=GitHubConfig)
    quality: QualityFilter = field(default_factory=QualityFilter)
    analyzer: AnalyzerConfig = field(default_factory=AnalyzerConfig)
    scheduler: SchedulerConfig = field(default_factory=SchedulerConfig)


@dataclass(frozen=True)
class EnvConfig:
    github_token: str | None
    gemini_api_key: str | None
    telegram_bot_token: str | None
    telegram_chat_id: str | None


_ENV_VARS = {
    "github_token": "GITHUB_TOKEN",
    "gemini_api_key": "GEMINI_API_KEY",
    "telegram_bot_token": "TELEGRAM_BOT_TOKEN",
    "telegram_chat_id": "TELEGRAM_CHAT_ID",
}


def load_env_config() -> EnvConfig:
    return EnvConfig(
        github_token=os.getenv("GITHUB_TOKEN"),
        gemini_api_key=os.getenv("GEMINI_API_KEY"),
        telegram_bot_token=os.getenv("TELEGRAM_BOT_TOKEN"),
        telegram_chat_id=os.getenv("TELEGRAM_CHAT_ID"),
    )


def ensure_env(env: EnvConfig, required_fields: tuple[str, ...]) -> None:
    missing = [
        _ENV_VARS[field]
        for field in required_fields
        if not getattr(env, field)
    ]
    if missing:
        missing_list = ", ".join(missing)
        raise ValueError(f"Missing required environment variables: {missing_list}")


config = AppConfig()
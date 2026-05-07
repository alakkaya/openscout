"""
analyzer.py

LLM-based issue analysis.
"""

from __future__ import annotations

import json
import logging
from typing import Any

from google import genai
from google.genai import types
from google.genai.errors import APIError

from config import AnalyzerConfig
from models import Issue, IssueAnalysis, ScoredIssue

log = logging.getLogger(__name__)

_SYSTEM_PROMPT = (
	"You are a senior open-source mentor. "
	"Return ONLY a valid JSON object. "
	"Do NOT include markdown code blocks, backticks, or any extra text. "
	"Start with { and end with }."
)

_USER_PROMPT_TEMPLATE = """
Analyze the GitHub issue and decide if it is solvable for a new contributor.

Return a JSON object with exactly these keys:
- complexity: integer 1-5
- estimated_hours: integer
- skills_needed: list of short strings
- why_solvable: short sentence
- warning: short sentence or null

Constraints:
- complexity must be 1-5
- estimated_hours must be 1-24
- keep why_solvable under 200 characters

Issue:
Title: {title}
Labels: {labels}
Comments: {comment_count}
Created: {created_at}
Repository: {repo_name} (stars: {stars})
Description: {repo_description}
Body:
{body}
""".strip()


class IssueAnalyzer:
    def __init__(self, config: AnalyzerConfig, client: genai.Client) -> None:
        self._config = config
        self._client = client

    def analyze(self, issues: list[Issue]) -> list[ScoredIssue]:
        scored: list[ScoredIssue] = []

        for issue in issues:
            analysis = self._analyze_issue(issue)
            if not analysis:
                continue
            if analysis.complexity > self._config.max_complexity_score:
                continue
            scored.append(ScoredIssue(issue=issue, analysis=analysis))

        scored.sort(
            key=lambda item: (
                item.analysis.complexity,
                item.analysis.estimated_hours,
                -item.issue.repository.stars,
            )
        )
        return scored[: self._config.top_issues_count]

    def _analyze_issue(self, issue: Issue) -> IssueAnalysis | None:
        prompt = _USER_PROMPT_TEMPLATE.format(
            title=issue.title,
            labels=", ".join(issue.labels) or "none",
            comment_count=issue.comment_count,
            created_at=issue.created_at.isoformat(),
            repo_name=issue.repository.name,
            stars=issue.repository.stars,
            repo_description=issue.repository.description or "",
            body=issue.body or "",
        )

        try:
            response = self._client.models.generate_content(
                model=self._config.model,
                contents=prompt,
                config=types.GenerateContentConfig(
                    system_instruction=_SYSTEM_PROMPT,
                    temperature=0.2,
                    max_output_tokens=self._config.max_tokens,
                    response_mime_type="application/json",
                ),
            )
        except APIError as exc:
            log.error("Analysis failed for issue %s: %s", issue.id, exc)
            return None

        log.debug("Response type: %s, content type: %s", type(response), type(getattr(response, "content", None)))
        content = _extract_text(response)
        if not content:
            log.warning("Empty response for issue %s (response=%s)", issue.id, response)
            return None
        
        data = _parse_json(content)
        if not data:
            log.warning("Invalid JSON for issue %s (got: %.100s)", issue.id, content)
            return None

        analysis = _build_issue_analysis(data)
        if not analysis:
            log.warning("Incomplete analysis for issue %s", issue.id)
        return analysis


def _extract_text(response: Any) -> str:
    content = getattr(response, "content", None)
    if isinstance(content, list):
        parts: list[str] = []
        for block in content:
            if isinstance(block, dict):
                parts.append(block.get("text", ""))
            else:
                parts.append(getattr(block, "text", ""))
        text = "".join(parts).strip()
    elif isinstance(content, str):
        text = content.strip()
    elif isinstance(response, dict):
        content = response.get("content")
        if isinstance(content, list):
            text = "".join(block.get("text", "") for block in content if isinstance(block, dict)).strip()
        elif isinstance(content, str):
            text = content.strip()
        else:
            text = ""
    else:
        text = ""
    
    # Strip markdown code blocks if present
    if text.startswith("```"):
        # Remove opening ``` (with optional language identifier like ```json)
        lines = text.split("\n")
        if lines[0].startswith("```"):
            lines = lines[1:]
        # Remove closing ```
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        text = "\n".join(lines).strip()
    
    return text


def _parse_json(text: str) -> dict[str, Any] | None:
    if not text:
        return None
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        start = text.find("{")
        end = text.rfind("}")
        if start == -1 or end == -1 or end <= start:
            return None
        try:
            return json.loads(text[start : end + 1])
        except json.JSONDecodeError:
            return None


def _build_issue_analysis(data: dict[str, Any]) -> IssueAnalysis | None:
    try:
        complexity = int(data.get("complexity", 0))
        estimated_hours = int(data.get("estimated_hours", 0))
    except (TypeError, ValueError):
        return None

    if complexity < 1 or complexity > 5:
        return None
    if estimated_hours < 1:
        estimated_hours = 1
    if estimated_hours > 24:
        estimated_hours = 24

    skills = data.get("skills_needed") or []
    if not isinstance(skills, list):
        skills = [str(skills)]
    skills = [str(skill).strip() for skill in skills if str(skill).strip()]

    why_solvable = str(data.get("why_solvable", "")).strip()
    warning = data.get("warning")
    warning_text = str(warning).strip() if warning else None

    if not why_solvable:
        return None

    return IssueAnalysis(
        complexity=complexity,
        estimated_hours=estimated_hours,
        skills_needed=skills,
        why_solvable=why_solvable,
        warning=warning_text,
    )

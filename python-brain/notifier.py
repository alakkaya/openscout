"""
notifier.py

Telegram notifications.
"""

import logging
from datetime import datetime

import httpx

from models import Issue, ScoredIssue

log = logging.getLogger(__name__)

_TELEGRAM_API = "https://api.telegram.org/bot{token}/{method}"

_COMPLEXITY_ICON = {1: "🟢", 2: "🟢", 3: "🟡", 4: "🟡", 5: "🟠"}
_DEFAULT_ICON = "🔴"


class TelegramNotifier:
    def __init__(
        self,
        bot_token: str,
        chat_id: str,
        http_client: httpx.Client | None = None,
    ) -> None:
        if not bot_token or not chat_id:
            raise ValueError("Telegram bot token and chat ID are required")
        self._bot_token = bot_token
        self._chat_id = chat_id
        self._client = http_client or httpx.Client(timeout=10)
        self._owns_client = http_client is None

    def __enter__(self) -> "TelegramNotifier":
        return self

    def __exit__(self, exc_type, exc, exc_tb) -> None:
        self.close()

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def send_digest(self, scored_issues: list[ScoredIssue]) -> bool:
        if not scored_issues:
            log.info("No issues to send.")
            return False

        payload = {
            "chat_id": self._chat_id,
            "text": _build_digest_message(scored_issues),
            "parse_mode": "HTML",
            "disable_web_page_preview": True,
        }

        try:
            data = self._post("sendMessage", payload)
            if data.get("ok"):
                log.info("Telegram message sent.")
                return True
            log.error("Telegram error: %s", data.get("description"))
            return False

        except httpx.RequestError as exc:
            log.error("Telegram connection error: %s", exc)
            return False

    def send_raw_issues(self, issues: list[Issue]) -> bool:
        if not issues:
            log.info("No issues to send.")
            return False

        payload = {
            "chat_id": self._chat_id,
            "text": _build_raw_issue_message(issues),
            "parse_mode": "HTML",
            "disable_web_page_preview": True,
        }

        try:
            data = self._post("sendMessage", payload)
            if data.get("ok"):
                log.info("Telegram message sent.")
                return True
            log.error("Telegram error: %s", data.get("description"))
            return False
        except httpx.RequestError as exc:
            log.error("Telegram connection error: %s", exc)
            return False

    def send_test_message(self) -> bool:
        payload = {
            "chat_id": self._chat_id,
            "text": "🔭 <b>OpenScout connection test</b>\n\nBot is working!",
            "parse_mode": "HTML",
        }
        try:
            data = self._post("sendMessage", payload)
            return bool(data.get("ok"))
        except httpx.RequestError:
            return False

    def get_bot_info(self) -> dict | None:
        try:
            data = self._client.get(self._telegram_url("getMe"), timeout=10).json()
            return data.get("result") if data.get("ok") else None
        except httpx.RequestError:
            return None

    def _telegram_url(self, method: str) -> str:
        return _TELEGRAM_API.format(token=self._bot_token, method=method)

    def _post(self, method: str, payload: dict) -> dict:
        response = self._client.post(self._telegram_url(method), json=payload)
        response.raise_for_status()
        return response.json()


def _complexity_icon(score: int) -> str:
    return _COMPLEXITY_ICON.get(score, _DEFAULT_ICON)


def _format_issue_block(index: int, scored: ScoredIssue) -> str:
    issue = scored.issue
    analysis = scored.analysis
    icon = _complexity_icon(analysis.complexity)
    skills = " · ".join(analysis.skills_needed[:3]) or "Belirtilmemis"

    lines = [
        f"{index}. <b>{issue.title}</b>",
        f"   📦 <code>{issue.repository.name}</code>  ⭐ {issue.repository.stars:,}",
        f"   {icon} Zorluk: {analysis.complexity}/5  🕐 ~{analysis.estimated_hours}h",
        f"   🔧 {skills}",
        f"   💡 {analysis.why_solvable}",
    ]

    if analysis.warning:
        lines.append(f"   ⚠️ {analysis.warning}")

    lines.append(f"   🔗 <a href='{issue.url}'>Issue'yu gor</a>")
    return "\n".join(lines)


def _build_digest_message(scored_issues: list[ScoredIssue]) -> str:
    today = datetime.now().strftime("%d %B %Y")
    separator = "\n" + "─" * 32 + "\n"

    header = f"🔭 <b>OpenScout — {today}</b>\nBugun icin {len(scored_issues)} katki firsati:\n"
    body = separator.join(
        _format_issue_block(i + 1, scored) for i, scored in enumerate(scored_issues)
    )
    footer = "\n\n─────────────────────────────────\n💬 Hangisine bakacaksin?\n🚀 <i>OpenScout</i>"

    return header + "\n" + body + footer


def _format_raw_issue_block(index: int, issue: Issue) -> str:
    labels = " · ".join(issue.labels[:3]) or "Etiket yok"
    return "\n".join(
        [
            f"{index}. <b>{issue.title}</b>",
            f"   📦 <code>{issue.repository.name}</code>  ⭐ {issue.repository.stars:,}",
            f"   💬 {issue.comment_count} yorum  🔖 {labels}",
            f"   🔗 <a href='{issue.url}'>Issue'yu gor</a>",
        ]
    )


def _build_raw_issue_message(issues: list[Issue]) -> str:
    today = datetime.now().strftime("%d %B %Y")
    visible_issues = issues[:5]
    separator = "\n" + "─" * 32 + "\n"

    header = f"🔭 <b>OpenScout — {today}</b>\nGitHub'dan gelen {len(visible_issues)} ham issue:\n"
    body = separator.join(
        _format_raw_issue_block(i + 1, issue) for i, issue in enumerate(visible_issues)
    )
    footer = "\n\n─────────────────────────────────\n💬 Bu akışı test ediyoruz\n🚀 <i>OpenScout</i>"

    return header + "\n" + body + footer
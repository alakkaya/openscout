"""
main.py

OpenScout entry point.
"""

from __future__ import annotations

import argparse
import logging

from google import genai
from dotenv import load_dotenv

from analyzer import IssueAnalyzer
from cache import IssueCache
from config import config, ensure_env, load_env_config
from github_client import GitHubClient
from notifier import TelegramNotifier
from scheduler import SchedulerService

log = logging.getLogger(__name__)


class OpenScoutApp:
    def __init__(
        self,
        github_client: GitHubClient,
        analyzer: IssueAnalyzer | None,
        notifier: TelegramNotifier,
        cache: IssueCache,
    ) -> None:
        self._github_client = github_client
        self._analyzer = analyzer
        self._notifier = notifier
        self._cache = cache

    def __enter__(self) -> "OpenScoutApp":
        return self

    def __exit__(self, exc_type, exc, exc_tb) -> None:
        self.close()

    def close(self) -> None:
        self._github_client.close()
        self._notifier.close()

    def run_pipeline(self) -> None:
        if self._analyzer is None:
            raise RuntimeError("Analyzer is required for the analyzed pipeline")

        issues = self._github_client.fetch_all_issues()
        if not issues:
            log.info("No issues returned from GitHub.")
            return

        issues = self._cache.filter_unseen(issues)
        if not issues:
            log.info("No unseen issues left after cache filter.")
            return

        scored = self._analyzer.analyze(issues)
        if not scored:
            log.info("No issues passed the analysis filter.")
            return

        if self._notifier.send_digest(scored):
            self._cache.mark_sent(scored)

    def run_raw_issue_pipeline(self) -> None:
        issues = self._github_client.fetch_all_issues()
        if not issues:
            log.info("No issues returned from GitHub.")
            return

        issues = self._cache.filter_unseen(issues)
        if not issues:
            log.info("No unseen issues left after cache filter.")
            return

        if self._notifier.send_raw_issues(issues):
            self._cache.mark_issues_sent(issues)


def _build_app(env) -> OpenScoutApp:
    github_client = GitHubClient(config.github, config.quality, env.github_token or "")
    analyzer = IssueAnalyzer(
        config.analyzer,
        genai.Client(api_key=env.gemini_api_key),
    )
    notifier = TelegramNotifier(
        bot_token=env.telegram_bot_token or "",
        chat_id=env.telegram_chat_id or "",
    )
    cache = IssueCache()
    return OpenScoutApp(github_client, analyzer, notifier, cache)


def _setup_logging(debug: bool) -> None:
    level = logging.DEBUG if debug else logging.INFO
    logging.basicConfig(
        level=level,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="OpenScout pipeline runner")
    parser.add_argument("--fetch-only", action="store_true", help="Fetch issues without analysis")
    parser.add_argument(
        "--send-raw-issues",
        action="store_true",
        help="Fetch issues and send them to Telegram without Gemini analysis",
    )
    parser.add_argument("--test-bot", action="store_true", help="Send a Telegram test message")
    parser.add_argument("--now", action="store_true", help="Run the pipeline immediately")
    parser.add_argument("--debug", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def main() -> int:
    args = _parse_args()
    load_dotenv()
    _setup_logging(args.debug)

    env = load_env_config()

    if args.test_bot:
        ensure_env(env, ("telegram_bot_token", "telegram_chat_id"))
        with TelegramNotifier(
            bot_token=env.telegram_bot_token or "",
            chat_id=env.telegram_chat_id or "",
        ) as notifier:
            ok = notifier.send_test_message()
        return 0 if ok else 1

    if args.fetch_only:
        ensure_env(env, ("github_token",))
        with GitHubClient(config.github, config.quality, env.github_token or "") as client:
            issues = client.fetch_all_issues()
        log.info("Fetched %d issues.", len(issues))
        return 0

    if args.send_raw_issues:
        ensure_env(env, ("github_token", "telegram_bot_token", "telegram_chat_id"))
        with GitHubClient(config.github, config.quality, env.github_token or "") as github_client:
            with TelegramNotifier(
                bot_token=env.telegram_bot_token or "",
                chat_id=env.telegram_chat_id or "",
            ) as notifier:
                cache = IssueCache()
                app = OpenScoutApp(github_client, None, notifier, cache)
                app.run_raw_issue_pipeline()
        return 0

    ensure_env(
        env,
        (
            "github_token",
            "gemini_api_key",
            "telegram_bot_token",
            "telegram_chat_id",
        ),
    )

    with _build_app(env) as app:
        if args.now:
            app.run_pipeline()
            return 0

        scheduler = SchedulerService(config.scheduler, app.run_pipeline)
        scheduler.start()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""
cache.py

SQLite-backed cache for sent issues.
"""

import logging
import os
import sqlite3
from contextlib import contextmanager
from datetime import datetime, timezone
from typing import Generator

from models import Issue, ScoredIssue

log = logging.getLogger(__name__)

_DEFAULT_DB_PATH = os.path.join(os.path.dirname(__file__), "..", "openscout.db")

_CREATE_TABLE = """
    CREATE TABLE IF NOT EXISTS seen_issues (
        id       TEXT PRIMARY KEY,
        title    TEXT NOT NULL,
        url      TEXT NOT NULL,
        sent_at  TEXT NOT NULL
    )
"""

class IssueCache:
    def __init__(self, db_path: str | None = None) -> None:
        self._db_path = db_path or _DEFAULT_DB_PATH
        self._init_db()

    @contextmanager
    def _connection(self) -> Generator[sqlite3.Connection, None, None]:
        conn = sqlite3.connect(self._db_path)
        conn.row_factory = sqlite3.Row
        try:
            yield conn
        finally:
            conn.close()

    def _init_db(self) -> None:
        with self._connection() as conn:
            conn.execute(_CREATE_TABLE)
            conn.commit()

    def filter_unseen(self, issues: list[Issue]) -> list[Issue]:
        """Return issues that have not been sent before."""
        with self._connection() as conn:
            ids = {row[0] for row in conn.execute("SELECT id FROM seen_issues")}
        return [issue for issue in issues if issue.id not in ids]

    def mark_sent(self, scored_issues: list[ScoredIssue]) -> None:
        """Persist sent issues to the cache."""
        if not scored_issues:
            return
        now = datetime.now(timezone.utc).isoformat()
        rows = [
            (s.issue.id, s.issue.title, s.issue.url, now)
            for s in scored_issues
        ]
        with self._connection() as conn:
            conn.executemany(
                "INSERT OR IGNORE INTO seen_issues (id, title, url, sent_at) VALUES (?,?,?,?)",
                rows,
            )
            conn.commit()
        log.info("Cached %d issues.", len(rows))

    def mark_issues_sent(self, issues: list[Issue]) -> None:
        """Persist sent issues to the cache."""
        if not issues:
            return
        now = datetime.now(timezone.utc).isoformat()
        rows = [
            (issue.id, issue.title, issue.url, now)
            for issue in issues
        ]
        with self._connection() as conn:
            conn.executemany(
                "INSERT OR IGNORE INTO seen_issues (id, title, url, sent_at) VALUES (?,?,?,?)",
                rows,
            )
            conn.commit()
        log.info("Cached %d issues.", len(rows))
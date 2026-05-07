"""
scheduler.py

APScheduler integration.
"""

import logging
from typing import Callable

from apscheduler.schedulers.blocking import BlockingScheduler
from apscheduler.triggers.cron import CronTrigger

from config import SchedulerConfig

log = logging.getLogger(__name__)


class SchedulerService:
    def __init__(self, config: SchedulerConfig, job: Callable[[], None]) -> None:
        self._config = config
        self._job = job

    def start(self) -> None:
        scheduler = BlockingScheduler(timezone=self._config.timezone)
        trigger = CronTrigger(
            hour=self._config.hour,
            minute=self._config.minute,
            timezone=self._config.timezone,
        )

        scheduler.add_job(
            self._job,
            trigger=trigger,
            id="openscout_digest",
            replace_existing=True,
            misfire_grace_time=self._config.misfire_grace_seconds,
        )

        log.info(
            "Scheduler started: %02d:%02d (%s)",
            self._config.hour,
            self._config.minute,
            self._config.timezone,
        )
        scheduler.start()

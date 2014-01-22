from __future__ import absolute_import
import time

from celery import Celery

app = Celery("cpcelery", broker="redis://", backend="redis://")

app.conf.update(
    CELERY_TASK_SERIALIZER='json',
    CELERY_ACCEPT_CONTENT=['json'],
    CELERY_RESULT_SERIALIZER='json',
    CELERYD_CONCURRENCY=2,
)

@app.task
def serviced_shell(cmd):
    with open('/opt/celery/var/int(time.time())', 'w') as f:
        f.write(str(cmd))


from __future__ import absolute_import
import time
import datetime
from dateutil import parser

from multiprocessing.util import Finalize
from celery import Celery
from celery import current_app
from celery.schedules import crontab
from celery.beat import Scheduler, ScheduleEntry
from pyes import TermQuery, ES
from socketIO_client import SocketIO
import socket
import json

REDIS_URL = "redis://"  # Default is localhost:6379, which is what we want
# Go directly to container gateway to hit CP elastic isvc. This will only
# be true while we can guarantee isvcs running on the same box.
ELASTIC_HOST = '172.17.42.1'
ELASTIC_URL = 'http://%s:9200' % ELASTIC_HOST


app = Celery("cpcelery", broker="redis://", backend="redis://")


app.conf.update(
    CELERY_TASK_SERIALIZER='json',
    CELERY_ACCEPT_CONTENT=['json'],
    CELERY_RESULT_SERIALIZER='json',
    CELERYD_CONCURRENCY=2,
    CELERYBEAT_SCHEDULER="cpcelery:ControlPlaneScheduler",
    CELERYBEAT_MAX_LOOP_INTERVAL=5
)

def log(data):
    data = json.dumps(data)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((ELASTIC_HOST, 5042))
    s.sendall(data)
    s.shutdown(socket.SHUT_WR)
    s.close()

class ServicedShell:
    def __init__(self):
        self.socket = None
        self.stdout = ""
        self.stderr = ""
        self.result = ""
    def onResult(self, *args):
        self.result = args[0]
        self.socket.disconnect()
    def onStdout(self, *args):
        for l in args:
            self.stdout += str(l)
    def onStderr(self, *args):
        for l in args:
            self.stderr += str(l)
    def run(self, service_id, command):        
        log({
            "command": command,
            "service_id": service_id,
            "status": "starting"
        })
        self.socket = SocketIO(ELASTIC_HOST, 50000)
        self.socket.on('result', self.onResult)
        self.socket.on('stdout', self.onStdout)        
        self.socket.on('stderr', self.onStderr)        
        self.socket.emit('process', {'Command': command, 'IsTTY': False, 'ServiceId': service_id, 'Envv': []})
        self.socket.wait()
        log({
            "command": command,
            "service_id": service_id,
            "status": "complete",
            "stdout": self.stdout,
            "stderr": self.stderr,
            "result": self.result
        })

@app.task()
def serviced_shell(service_id, command):
    s = ServicedShell()
    s.run(service_id, command)

class ControlPlaneScheduleEntry(ScheduleEntry):

    task = "cpcelery.serviced_shell"

    def __init__(self, svc_model=None, task_model=None):
        self.app = current_app._get_current_object()

        self.svc_model = svc_model
        self.task_model = task_model

        self.options = {}
        self.name = task_model.Name
        self.args = [svc_model.Id, task_model.Command]
        self.schedule = crontab(*task_model.Schedule.split()) 
        self.total_run_count = task_model.TotalRunCount or 0
        task_model.LastRunAt = task_model.LastRunAt or "0001-01-01T00:00:00Z"
        if isinstance(task_model.LastRunAt, basestring):
            task_model.LastRunAt = parser.parse(task_model.LastRunAt)
        self.last_run_at = task_model.LastRunAt

    def is_due(self):
        result = super(ControlPlaneScheduleEntry, self).is_due()
        return result

    def save(self):
        # Object may not be synchronized, so only change the fields we care
        # about. Get a new copy of the service.
        meta = self.svc_model._meta
        svc = meta.connection.get(meta.index, meta.type, meta.id)
        for task in svc.Tasks:
            # Iterate. Only way to get our task, sadly. Pretty cheap tho.
            if task.Name == self.name:
                # Store date format we know everybody can parse in Elastic
                # (Everybody is us, Elastic and control plane model)
                task.LastRunAt = self.last_run_at.isoformat() + 'Z'
                task.TotalRunCount = self.total_run_count
                # Save the service with our changes
                svc.save()
                # Use the new model with potentially updated info
                self.task_model = task
                self.svc_model = svc
                return

    def _next_instance(self):
        self.task_model.LastRunAt = self.app.now()
        self.task_model.TotalRunCount += 1
        return self.__class__(self.svc_model, self.task_model)

    __next__ = next = _next_instance


class ControlPlaneScheduler(Scheduler):

    Entry = ControlPlaneScheduleEntry

    _elastic = None
    _schedule = None
    _last_timestamp = None
    _initial_read = False

    def __init__(self, *args, **kwargs):
        self._dirty = set()
        # We have to wait for the elastic container to start or things go
        # sideways.
        # TODO: Check status properly somehow (straight HTTP request, perhaps)
        time.sleep(30)
        self._elastic = ES(ELASTIC_URL, max_retries=100)
        self._finalize = Finalize(self, self.sync, exitpriority=5)
        super(ControlPlaneScheduler, self).__init__(*args, **kwargs)

    def setup_schedule(self):
        self.install_default_entries(self.schedule)
        self.update_from_dict(self.app.conf.CELERYBEAT_SCHEDULE)

    def reserve(self, entry):
        """
        This is called when a new instance of a task is scheduled to run. Hook
        in here so we can avoid saving updates to tasks that have none.
        """
        new_entry = Scheduler.reserve(self, entry)
        # Add to a list of what has changed. Store by name since the entry
        # itself may be a different instance by the time we get to it.
        self._dirty.add(new_entry.name)
        return new_entry

    def update_from_dict(self, dict_):
        """
        Copied from django-celery scheduler.
        """
        s = {}
        for name, entry in dict_.items():
            try:
                s[name] = self.Entry.from_entry(name, **entry)
            except Exception, exc:
                self.logger.error(
                    "Couldn't add entry %r to database schedule: %r. "
                    "Contents: %r" % (name, exc, entry))
        self._schedule.update(s)

    def all_as_schedule(self):
        """
        Get the current schedule comprising entries built from Elastic data.
        """
        self.logger.debug("ControlPlaneScheduler: Fetching database schedule")
        entries = {}
        for svc in self._elastic.search(TermQuery("_type", "service")):
            for task in svc.Tasks or ():
                entry = self.Entry(svc_model=svc, task_model=task)
                entries[entry.name] = entry
        return entries

    @property
    def schedule(self):
        """
        """
        self.sync()
        self._schedule = self.all_as_schedule()
        return self._schedule

    def sync(self):
        """
        Save off whatever tasks have been updated with run time and count.
        """
        while self._dirty:
            name = self._dirty.pop()
            self._schedule[name].save()

    @property
    def info(self):
        return '    . db -> {elastic_url}'.format(elastic_url=ELASTIC_URL)


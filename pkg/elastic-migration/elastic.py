import json
import re
import logging
from optparse import OptionParser
from ConfigParser import ConfigParser

from elasticsearch1 import Elasticsearch as ES1x
from elasticsearch import Elasticsearch as ES7x

from elasticsearch1.helpers import scan
from elasticsearch.helpers import bulk



class BaseUtil:

    def __init__(self, file_name, config_name):
        self.config = ConfigParser()
        self.config.read(config_name)
        self.fname = file_name

        log_formatter = logging.Formatter("%(asctime)s [%(levelname)s]  %(message)s")

        self.log = logging.getLogger()
        self.log.setLevel(logging.INFO)

        file_handler = logging.FileHandler("{0}/{1}.log".format(self.config.get('DEFAULT', 'log_path'),
                                                                self.config.get('DEFAULT', 'log_file_name')))
        file_handler.setFormatter(log_formatter)
        self.log.addHandler(file_handler)

        console_handler = logging.StreamHandler()
        console_handler.setFormatter(log_formatter)
        self.log.addHandler(console_handler)


class ExportUtil(BaseUtil):

    def __call__(self, *args, **kwargs):
        with open(self.fname, "w") as f:
            if "logstash" in kwargs and kwargs["logstash"]:
                f.write(self.__get_json_data_logstash())
            else:
                f.write(self.__get_json_data())
        self.log.info("Export is done")

    def __get_json_data(self):

        self.log.info("Connecting to the elastic cluster by address  %s", self.config.get('DEFAULT', 'elastic_host_port'))
        es = ES1x([self.config.get('DEFAULT', 'elastic_host_port')], use_ssl=False)

        self.types = re.sub(r"[\n\t\s]*", "", self.config.get('DEFAULT', 'types')).split(",")

        match_all = {"size": self.config.getint('DEFAULT', "query_batch_size"), "query": {"match_all": {}}}

        results = dict((k, []) for k in self.types)

        for current_type in self.types:
            self.log.info("Starting fetching %s type", current_type)
            data = es.search(index=self.config.get('DEFAULT', 'index'), doc_type=current_type, body=match_all, scroll="2s")
            sid = data['_scroll_id']
            scroll_size = len(data['hits']['hits'])
            self.log.info("Fetched %d doc(s)", scroll_size)

            while scroll_size > 0:
                results[current_type].extend([
                    dict(id=item["_id"], source=item["_source"])
                    for item in data['hits']['hits']
                ])

                data = es.scroll(scroll_id=sid, scroll='2s')
                sid = data['_scroll_id']
                scroll_size = len(data['hits']['hits'])
            self.log.info("Finished fetching %s type", current_type)

        self.log.info("Dumping all result to json")
        return json.dumps(results, indent=2, sort_keys=True)

    def __get_json_data_logstash(self):
        self.log.info("Connecting to the elastic cluster by address  %s", self.config.get('DEFAULT', 'elastic_logstash_host_port'))
        es = ES1x([self.config.get('DEFAULT', 'elastic_logstash_host_port')],
                   timeout=int(self.config.get('DEFAULT', 'timeout')),
                   use_ssl=False,
                   retry_on_timeout=True)
        _, data = es.transport.perform_request('GET', '/_all/_mapping')
        actions = []
        indices_type = []

        for index, value in data.items():
            for type in value.get("mappings"):
                indices_type.append((index, type))

        match_all = {"size": self.config.getint('DEFAULT', "query_batch_size"), "query": {"match_all": {}}}

        for index, type in indices_type:
            self.log.info("Starting fetching index: %s; type: %s", index, type)

            data = scan(es, query=match_all, size=self.config.getint('DEFAULT', "query_batch_size"),
                        index=index, doc_type=type)

            for item in data:
                item["_source"].update({"type": type})
                actions.append({'_op_type': 'create',
                                '_index': index,
                                '_id': item["_id"],
                                '_source': item["_source"]})
            self.log.info("Finished fetching index: %s; type: %s", index, type)

        self.log.info("Dumping all result to json")
        return json.dumps(actions, indent=2, sort_keys=True)


class ImportUtil(BaseUtil):

    def __call__(self, *args, **kwargs):
        with open(self.fname) as f:
            if "logstash" in kwargs and kwargs["logstash"]:
                self.__set_json_data_logstash(json.load(f))
            else:
                self.__set_json_data(json.load(f))
        self.log.info("Import is done")

    def __set_json_data(self, data):
        self.log.info("Connecting to the elastic cluster by address  %s", self.config.get('DEFAULT', 'elastic_host_port'))
        es = ES7x([self.config.get('DEFAULT', 'elastic_host_port')],
                  timeout=int(self.config.get('DEFAULT', 'timeout')),
                  use_ssl=False, retry_on_timeout=True)
        self.types = re.sub(r"[\n\t\s]*", "", self.config.get('DEFAULT', 'types')).split(",")
        index = self.config.get('DEFAULT','index')

        with open("%s/%s.json" % (self.config.get('DEFAULT','index_config_path'), index)) as f:
            index_config = json.load(f)

        es.indices.create(index, body=index_config)
        self.log.info("Created new elastic version index '%s' for importing data" % index)

        for current_type, values in data.items():
            self.log.info("Starting adding data of %s type", current_type)
            for item in values:
                item["source"].update({"type": current_type})
                es.create(index, id="%s-%s" % (item["id"], current_type), body=item["source"])
            self.log.info("Created %d doc(s)", len(values))
            self.log.info("Finished adding data of %s type", current_type)

    def __set_json_data_logstash(self, actions):
        self.log.info("Connecting to the elastic cluster by address  %s", self.config.get('DEFAULT', 'elastic_logstash_host_port'))
        es = ES7x([self.config.get('DEFAULT', 'elastic_logstash_host_port')], use_ssl=False)
        bulk(es, actions, chunk_size=self.config.get('DEFAULT', 'chunk_size'),
                          max_retries=int(self.config.get('DEFAULT', 'max_retries')),
                          max_chunk_bytes=int(self.config.get('DEFAULT', 'max_chunk_bytes')) * 1024 * 1024,
                          stats_only=True,
                          timeout="%ss" % self.config.get('DEFAULT', 'timeout'))

def main():
    parser = OptionParser()

    parser.add_option("-f", "--file", dest="filename", default="data.json",
                      help="export to / import from FILE", metavar="FILE")

    parser.add_option("-c", "--config", dest="config_name", default="es_cluster.ini",
                      help="config file of elastic cluster", metavar="FILE")

    parser.add_option("-e", "--export", action="store_true", dest="export",
                      help="make an export from old elastic serviced")

    parser.add_option("-E", "--export-logstash", action="store_true", dest="ls_export",
                      help="make an export from old elastic logstash")

    parser.add_option("-i", "--import", action="store_true", dest="import",
                      help="make an import to new elastic serviced")

    parser.add_option("-I", "--import-logstash", action="store_true", dest="ls_import",
                      help="make an import to new elastic logstash")

    (options, _) = parser.parse_args()

    if options.export:
        ExportUtil(options.filename, options.config_name)()
    elif options.ls_export:
        ExportUtil(options.filename, options.config_name)(logstash=True)
    elif getattr(options, "import"):
        ImportUtil(options.filename, options.config_name)()
    elif options.ls_import:
        ImportUtil(options.filename, options.config_name)(logstash=True)


if __name__ == "__main__":
    main()

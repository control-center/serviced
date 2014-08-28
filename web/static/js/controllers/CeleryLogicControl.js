function CeleryLogControl($scope, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "celerylog";
    $scope.page = 1;
    $scope.pageCount = 1;

    $scope.logs = buildTable('StartTime', [
        { id: 'JobID', name: 'celery_tbl_jobid' },
        { id: 'Command', name: 'celery_tbl_command' },
        { id: 'StartTime', name: 'celery_tbl_starttime' },
        { id: 'EndTime', name: 'celery_tbl_endtime' },
        { id: 'ExitCode', name: 'celery_tbl_exitcode' },
    ]);

    $scope.client = new elasticsearch.Client({host: 'localhost:9200'});

    $scope.commandQuery = function() {
        return {
            body: {
                size: 16,
                from: ($scope.page - 1) * 16,
                sort: [
                    {
                        "@timestamp": {
                            order: "desc"
                        }
                    }
                ],
                query: {
                    filtered: {
                        query: {
                            match_all: {}
                        },
                        filter: {
                            term: {
                                "logtype": "command"
                            }
                        }
                    }
                }
            }
        };
    };

    $scope.jobQuery = function(jobid, size) {
        return {
            body: {
                size: size,
                sort: [
                    {
                        "@timestamp": {
                            order: "asc"
                        }
                    }
                ],
                query: {
                    filtered: {
                        query: {
                            match_all: {}
                        },
                        filter: {
                            term: {
                                "jobid.raw": jobid
                            }
                        }
                    }
                }
            }
        };
    };

    $scope.exitQuery = function(jobids) {
        return {
            size: 32,
            body: {
                query: {
                    filtered: {
                        query: {
                            match_all: {}
                        },
                        filter: {
                            and: [
                                {
                                    terms: {
                                        "jobid.raw": jobids
                                    }
                                },
                                {
                                    term: {
                                        "logtype.raw": "exitcode"
                                    }
                                }
                            ]
                        }
                    }
                }
            }
        };
    };

    $scope.buildPage = function() {
        var jobids = [];
        var jobrecords = [];
        var jobmapping = {};
        // Get a count of job start and finish logs.
        $scope.client.search($scope.commandQuery()).then(function(body) {
            $scope.pageCount = Math.max(1, Math.ceil(body.hits.total/16));
            $scope.leftDisabled = false;
            $scope.rightDisabled = false;
            if ($scope.page == 1) {
                $scope.leftDisabled = true;
            }
            if ($scope.page == $scope.pageCount) {
                $scope.rightDisabled = true;
            }
            // Create a list of jobids for the command log lines.
            for (var i = 0; i < body.hits.hits.length; i++) {
                var hit = body.hits.hits[i]._source;
                jobids.push(hit.jobid);
                var record = {jobid: hit.jobid};
                record.jobid_short = record.jobid.slice(0,8) + '...';
                record.command = hit.command;
                record.starttime = hit['@timestamp'];
                jobrecords.push(record);
                jobmapping[hit.jobid] = record;
            }
            // Get all the exitcodes associated with the jobids and fill out the records.
            $scope.client.search($scope.exitQuery(jobids)).then(function(body) {
                for (var i = 0; i < body.hits.hits.length; i++) {
                    var hit = body.hits.hits[i]._source;
                    jobmapping[hit.jobid].exitcode = hit.exitcode;
                    jobmapping[hit.jobid].endtime = hit['@timestamp'];
                }
                $scope.logs.data = jobrecords;
                $scope.$apply();
                $("abbr.timeago").timeago();
            });
        });
    };

    $scope.pageLeft = function() {
        $scope.page--;
        $scope.buildPage();
    };

    $scope.pageRight = function() {
        $scope.page++;
        $scope.buildPage();
    };

    $scope.click_jobid = function(jobid) {
        $scope.client.search($scope.jobQuery(jobid, 0)).then(function(count) {
            $scope.client.search($scope.jobQuery(jobid, count.hits.total)).then(function(body) {
                $scope.loglines = "";
                for (var i = 0; i < body.hits.hits.length; i++) {
                    var hit = body.hits.hits[i]._source;
                    if (hit.logtype == "command") {
                        $scope.loglines += hit.command + "\n";
                    }
                    else if (hit.logtype == "stdout") {
                        $scope.loglines += hit.stdout;
                    }
                    else if (hit.logtype == "stderr") {
                        $scope.loglines += hit.stderr;
                    }
                    else if (hit.logtype == "exitcode") {
                        $scope.loglines += hit.exitcode;
                    }
                }
                $scope.$apply();
                $modalService.create({
                    templateUrl: "view-celery-log.html",
                    model: $scope,
                    title: "title_log",
                    bigModal: true,
                    actions: [
                        {
                            role: "cancel",
                            classes: "btn-default",
                            label: "close"
                        }
                    ]
                });
            });
        });
    };

    $scope.buildPage();

}

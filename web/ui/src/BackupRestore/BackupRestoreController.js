/* BackupRestoreController
 * Lists existing backups and allows creation
 * of new backups.
 */
(function() {
    'use strict';

    controlplane.controller("BackupRestoreController", ["$scope", "$routeParams", "$notification", "$translate", "resourcesFactory", "authService", "$modalService",
    function($scope, $routeParams, $notification, $translate, resourcesFactory, authService, $modalService) {
        // Ensure logged in
        authService.checkLogin($scope);

        // localization messages
        var BACKUP_RUNNING = $translate.instant("backup_running"),
            BACKUP_COMPLETE = $translate.instant("backup_complete"),
            RESTORE_RUNNING = $translate.instant("restore_running"),
            RESTORE_COMPLETE = $translate.instant("restore_complete"),
            ERROR = $translate.instant("error");

        $scope.createBackup = function () {

            let modalScope = $scope.$new(true);

            $modalService.create({
                templateUrl: "add-backup.html",
                model: modalScope,
                title: "backup_create",
                actions: [
                    {
                        role: "cancel"
                    }, {
                        role: "ok",
                        label: "backup_create",
                        classes: "btn-primary submit backup-ok",
                        action: function () {
                            var notification = $notification.create("Backup").updateStatus(BACKUP_RUNNING).show(false);

                            // TODO - when the server switches to broadcast instead of
                            // channel. this can be greatly simplified
                            resourcesFactory.createBackup().success(function checkFirstStatus() {
                                // recursively check if a valid status has been pushed into
                                // the pipe. if not, shake yourself off and try again. try again.
                                resourcesFactory.getBackupStatus().success(function (data) {
                                    // no status has been pushed, so check again
                                    if (data.Detail === "") {
                                        checkFirstStatus();

                                        // a valid status has been pushed, so
                                        // start the usual poll cycle
                                    } else {
                                        pollBackupStatus(notification);
                                    }

                                })
                                    .error(function (data, status) {
                                        backupRestoreError(notification, data.Detail, status);
                                    });
                            })
                                .error(function (data, status) {
                                    backupRestoreError(notification, data.Detail, status);
                                });

                            this.close();
                        }
                    }
                ],
                onShow: function () {

                    this.disableAction("backup-ok");
                    modalScope.backupCheck = {
                        BackupPath: "",
                        AvailableString: "",
                        EstimatedString: "",
                        AllowBackup: true,
                        pct: 0,
                        mcKraken: "",
                        ready: "",
                    };

                    resourcesFactory.getBackupCheck()
                        .success((data, status) => {

                            let pct = data.EstimatedBytes / data.AvailableBytes * 100;
                            pct = Math.min(100, pct); // don't break out of box
                            pct = Math.max(1, pct);   // show _something_

                            modalScope.backupCheck.pct = pct;
                            modalScope.backupCheck.BackupPath = data.BackupPath;
                            modalScope.backupCheck.AvailableString = data.AvailableString;
                            modalScope.backupCheck.EstimatedString = data.EstimatedString;
                            modalScope.backupCheck.mcKraken = "disappear";

                            if (data.AllowBackup) {
                                modalScope.backupCheck.ready = "ready";
                                this.enableAction("backup-ok");
                            } else {
                                modalScope.backupCheck.ready = "ready danger";
                            }
                        })

                        .error((data, status) => {
                            // do something
                        });

                }
            });
        };

        $scope.restoreBackup = function(filename){
            $modalService.create({
                template: $translate.instant("confirm_start_restore"),
                model: $scope,
                title: "restore",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "restore",
                        classes: "btn-danger",
                        action: function(){
                            var notification = $notification.create("Restore").updateStatus(RESTORE_RUNNING).show(false);

                            // TODO - when the server switches to broadcast instead of
                            // channel. this can be greatly simplified
                            resourcesFactory.restoreBackup(filename).success(function checkFirstStatus(){
                                // recursively check if a valid status has been pushed into
                                // the pipe. if not, shake yourself off and try again. try again.
                                resourcesFactory.getRestoreStatus().success(function(data){
                                    // no status has been pushed, so check again
                                    if(data.Detail === ""){
                                       checkFirstStatus();

                                    // a valid status has been pushed, so
                                    // start the usual poll cycle
                                    } else {
                                        notification.updateStatus(data.Detail);
                                        pollRestoreStatus(notification);
                                    }

                                })
                                .error(function(data, status){
                                    backupRestoreError(notification, data.Detail, status);
                                });
                            })
                            .error(function(data, status){
                                backupRestoreError(notification, data.Detail, status);
                            });

                            this.close();
                        }
                    }
                ]
            });
        };

        function pollBackupStatus(notification){
            resourcesFactory.getBackupStatus().success(function(data){

                if(data.Detail === ""){
                    notification.updateStatus(BACKUP_COMPLETE);
                    notification.success(false);
                    getBackupFiles();
                    return;
                }
                else if (data.Detail !== "timeout"){
                    notification.updateStatus(data.Detail);
                }

                // poll again
                setTimeout(function(){
                    pollBackupStatus(notification);
                }, 3000);

            })
            .error(function(data, status){
                backupRestoreError(notification, data.Detail, status);
            });
        }

        function pollRestoreStatus(notification){
            resourcesFactory.getRestoreStatus().success(function(data){

                // all done!
                if(data.Detail === ""){
                    notification.updateStatus(RESTORE_COMPLETE);
                    notification.success(false);
                    return;

                // something neato has happened. lets show it.
                } else if(data.Detail !== "timeout"){
                    notification.updateStatus(data.Detail);
                }

                // poll again
                setTimeout(function(){
                    pollRestoreStatus(notification);
                }, 3000);

            })
            .error(function(data, status){
                backupRestoreError(notification, data.Detail, status);
            });

        }

        function backupRestoreError(notification, data, status){
            notification.updateTitle(ERROR +" "+ status);
            notification.updateStatus(data);
            notification.error();
        }

        function getBackupFiles(){
            resourcesFactory.getBackupFiles().success(function(data){
                $scope.backupFiles = data;

                $scope.$emit("ready");
            });
        }

        function init(){
            $scope.name = "backuprestore";
            $scope.params = $routeParams;
            $scope.breadcrumbs = [{ label: 'breadcrumb_backuprestore', itemClass: 'active' }];

            $scope.backupTable = {
                sorting: {
                    full_path: "asc"
                },
                searchColumns: ['full_path']
            };

            //load backup files
            getBackupFiles();

            // poll for backup files
            resourcesFactory.registerPoll("running", getBackupFiles, 5000);
        }

        init();

        $scope.$on("$destroy", function(){
            resourcesFactory.unregisterAllPolls();
        });
    }]);
})();

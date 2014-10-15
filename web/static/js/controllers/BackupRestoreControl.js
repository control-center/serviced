function BackupRestoreControl($scope, $routeParams, $notification, $translate, resourcesService, authService, $modalService) {
    // Ensure logged in
    authService.checkLogin($scope);
    $scope.name = "backuprestore";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [{ label: 'breadcrumb_backuprestore', itemClass: 'active' }];

    //load backup files
    resourcesService.get_backup_files(function(data){
        $scope.backupFiles = data;
    });

    // localization messages
    var BACKUP_RUNNING = $translate.instant("backup_running"),
        BACKUP_COMPLETE = $translate.instant("backup_complete"),
        RESTORE_RUNNING = $translate.instant("restore_running"),
        RESTORE_COMPLETE = $translate.instant("restore_complete"),
        ERROR = $translate.instant("error");

    $scope.createBackup = function(){
        $modalService.create({
            template: $translate.instant("confirm_start_backup"),
            model: $scope,
            title: "backup_create",
            actions: [
                {
                    role: "cancel"
                },{
                    role: "ok",
                    label: "backup_create",
                    action: function(){
                        var notification = $notification.create("Backup").updateStatus(BACKUP_RUNNING).show(false);

                        // TODO - when the server switches to broadcast instead of
                        // channel. this can be greatly simplified
                        resourcesService.create_backup(function checkFirstStatus(){
                            // recursively check if a valid status has been pushed into
                            // the pipe. if not, shake yourself off and try again. try again.
                            resourcesService.get_backup_status(function(data){
                                if(data.Detail === ""){
                                   checkFirstStatus();
                                } else {
                                    notification.updateStatus(data.Detail);
                                    getBackupStatus(notification);
                                }
                            }, function(data, status){
                                backupRestoreError(notification, data.Detail, status);
                            });
                        }, function(data, status){
                            backupRestoreError(notification, data.Detail, status);
                        });

                        this.close();
                    }
                }
            ]
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
                        resourcesService.restore_backup(filename, function checkFirstStatus(){
                            // recursively check if a valid status has been pushed into
                            // the pipe. if not, shake yourself off and try again. try again.
                            resourcesService.get_restore_status(function(data){
                                if(data.Detail === ""){
                                   checkFirstStatus();
                                } else {
                                    notification.updateStatus(data.Detail);
                                    getRestoreStatus(notification);
                                }
                            }, function(data, status){
                                backupRestoreError(notification, data.Detail, status);
                            });
                        }, function(data, status){
                            backupRestoreError(notification, data.Detail, status);
                        });

                        this.close();
                    }
                }
            ]
        });
    };

    function getBackupStatus(notification){
        resourcesService.get_backup_status(function(data){

            if(data.Detail === ""){
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });

                notification.updateStatus(BACKUP_COMPLETE);
                notification.success(false);
                return;
            }
            else if (data.Detail !== "timeout"){
                notification.updateStatus(data.Detail);
            }

            // poll again
            setTimeout(function(){
                getBackupStatus(notification);
            }, 1);

        }, function(data, status){
            backupRestoreError(notification, data.Detail, status);
        });
    }

    function getRestoreStatus(notification){
        resourcesService.get_restore_status(function(data){

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
                getRestoreStatus(notification);
            }, 1);

        }, function(data, status){
            backupRestoreError(notification, data.Detail, status);
        });

    }

    function backupRestoreError(notification, data, status){
        notification.updateTitle(ERROR +" "+ status);
        notification.updateStatus(data);
        notification.error();
    }
}

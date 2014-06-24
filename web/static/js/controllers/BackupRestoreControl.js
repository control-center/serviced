function BackupRestoreControl($scope, $routeParams, $notification, resourcesService, authService, $translate, $templateCache) {

    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "backupRestoreControl";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [{ label: 'breadcrumb_backuprestore', itemClass: 'active' }];

    //load backup files
    resourcesService.get_backup_files(function(data){
        $scope.backupFiles = data;
    });

    // localization messages
    var BACKUP_RUNNING = $translate("backup_running"),
        BACKUP_COMPLETE = $translate("backup_complete"),
        RESTORE_RUNNING = $translate("restore_running"),
        RESTORE_COMPLETE = $translate("restore_complete"),
        ERROR = $translate("error");

    $scope.createBackup = function(){

        var notification = $notification.create().updateStatus(BACKUP_RUNNING).show(false);

        resourcesService.create_backup(function(data){
            setTimeout(function(){
                getBackupStatus(notification);
            }, 1);
        });
    };

    $scope.restoreBackup = function(filename){

        var notification = $notification.create().updateStatus(RESTORE_RUNNING).show(false);

        resourcesService.restore_backup(filename, function(data){
            setTimeout(function(){
                getRestoreStatus(notification);
            }, 1);
        });
    };

    function getBackupStatus(notification){
        resourcesService.get_backup_status(function(data){

            if(data.Detail === ""){
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });

                notification.updateStatus(BACKUP_COMPLETE);
                notification.success();
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
                notification.updateTitle(ERROR +" "+ status);
                notification.updateStatus(data.Detail);
                notification.error();
        });
    }

    function getRestoreStatus(notification){
        resourcesService.get_restore_status(function(data){

            // all done!
            if(data.Detail === ""){
                notification.updateStatus(RESTORE_COMPLETE);
                notification.success();
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
            notification.updateTitle(ERROR +" "+ status);
            notification.updateStatus(data.Detail);
            notification.error();
        });

    }
}
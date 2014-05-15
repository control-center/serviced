function BackupRestoreControl($scope, $routeParams, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "backupRestoreControl";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [{ label: 'breadcrumb_backuprestore', itemClass: 'active' }];

    //load backup files
    resourcesService.get_backup_files(function(data){
        $scope.backupFiles = data;
    });

    $scope.createBackup = function(){
        $('#backupInfo').show({
            duration: 200,
            easing: "linear"
        });
        resourcesService.create_backup(function(data){
            setTimeout(getBackupStatus, 1);
        });
    };

    $scope.restoreBackup = function(filename){
        $('#workingModal').modal('show');
        resourcesService.restore_backup(filename, function(data){
            $('#workingModal').modal('hide');
        });
    };

    function getBackupStatus(){
        resourcesService.get_backup_status(function(data){
            if(data.Detail != ""){
                if(data.Detail != "timeout"){
                    $("#backupStatus").html(data.Detail);
                }
                setTimeout(getBackupStatus, 1);
            }else{
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });
                $("#backupInfo").hide({
                    duration: 200,
                    easing: "linear"
                });
            }
        });
    }
}
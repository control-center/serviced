function BackupRestoreControl($scope, $routeParams, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "backupRestoreControl";
    $scope.params = $routeParams;
    $scope.breadcrumbs = [{ label: 'breadcrumb_backuprestore', itemClass: 'active' }];
    $scope.backupFiles = ["/tmp/backup-2014-05-09-153412.tgz"];

    $scope.createBackup = function(){
        $('#workingModal').modal('show');
        resourcesService.create_backup(function(data){
            $scope.backupFiles.push(data.Detail);
            $('#workingModal').modal('hide');
        });
    };

    $scope.restoreBackup = function(filename){
        $('#workingModal').modal('show');
        resourcesService.restore_backup(filename, function(data){
            $('#workingModal').modal('hide');
        });
    };
}

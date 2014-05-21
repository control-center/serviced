function BackupRestoreControl($scope, $routeParams, resourcesService, authService, $translate, $templateCache) {

    // cache reference to backup notification
    var backupNotification;

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

        // if existing backup, dont do anything?

        backupNotification = createBackupNotification("Backup Running");
        $("#backup_data").before(backupNotification);
        backupNotification.show("fast");

        resourcesService.create_backup(function(data){
            setTimeout(getBackupStatus, 1);
        });
    };

    $scope.restoreBackup = function(filename){
        $('#restoreInfo').show("fast");
        resourcesService.restore_backup(filename, function(data){
            setTimeout(getRestoreStatus, 1);
        });
    };

    function getBackupStatus(){
        resourcesService.get_backup_status(function(data){
            if(data.Detail !== ""){
                if(data.Detail !== "timeout"){
                    backupNotification.find(".backupStatus").html(data.Detail);
                }
                setTimeout(getBackupStatus, 1);
            }else{
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });

                console.log($scope);
                successifyNotification(backupNotification);
            }
        });
    }

    function getRestoreStatus(){
        resourcesService.get_restore_status(function(data){
            if(data.Detail !== ""){
                if(data.Detail !== "timeout"){
                    $("#restoreStatus").html(data.Detail);
                }
                setTimeout(getRestoreStatus, 1);
            }else{
                $("#restoreInfo").hide({
                    duration: 200,
                    easing: "linear"
                });
            }
        });
    }

    function createBackupNotification(message){
        var notify = $($templateCache.get("backupInfoNotification.html"));
        notify.find(".backupRunning").text(message);
        return notify;
    }

    // TODO - make this not terrible
    function successifyNotification(notification){
        // change notification color, icon, text, etc
        notification.removeClass("bg-info").addClass("bg-success");
        notification.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-ok-sign");
        // TODO - localization of text
        notification.find(".backupRunning").text("Backup Complete");
        notification.find(".backupStatus").html("");

        // show close button and make it active
        notification.find(".close").show().off().on("click", function(e){
            notification.hide("fast", function(){
                notification.remove();
            });
        });
    }
}
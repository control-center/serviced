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

        /*JsDbg*/debugger;// TODO - localization of message
        backupNotification = new Notification("Backup Running");
        $("#backup_data").before(backupNotification);
        backupNotification.show();

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
                    backupNotification.updateStatus(data.Detail);
                }
                setTimeout(getBackupStatus, 1);
            }else{
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });

                // TODO - localization of message
                backupNotification.successify("Backup Complete");
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


    // NOTE: this might not be the angular way to do this.
    // plus this notification should eventually be reusable
    // outside of just backup/restore
    
    /**
     * Notification
     * creates a notification. fun!
     */
    function Notification(title){
        this.$el = $($templateCache.get("backupInfoNotification.html"));
        this.$status = this.$el.find(".backupStatus");
        this.$title = this.$el.find(".backupRunning");
        this.updateTitle(title);
    }
    Notification.prototype = {
        constructor: Notification,

        // makes notification successy
        successify: function(title, msg){
            // change notification color, icon, text, etc
            this.$el.removeClass("bg-info").addClass("bg-success");
            this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-ok-sign");
            
            this.updateTitle(title);
            this.updateStatus(msg);

            // show close button and make it active
            this.$el.find(".close").show().off().on("click", function(e){
                this.$el.hide("fast", function(){
                    this.$el.remove();
                }.bind(this));
            }.bind(this));
        },
        // updates the status message (the smaller text)
        updateStatus: function(msg){
            this.$status.html(msg || "");
        },
        // updates the notification title (larger text)
        updateTitle: function(title){
            this.$title.text(title || "");
        },
        show: function(){
            this.$el.show("fast");
        }
    };
}
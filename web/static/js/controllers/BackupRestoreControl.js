function BackupRestoreControl($scope, $routeParams, resourcesService, authService, $translate, $templateCache) {

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
        CANNOT_BEGIN_BACKUP = $translate("cannot_begin_backup"),
        RESTORE_RUNNING = $translate("restore_running"),
        RESTORE_COMPLETE = $translate("restore_complete"),
        CANNOT_BEGIN_RESTORE = $translate("cannot_begin_restore"),
        ERROR = $translate("error");

    $scope.createBackup = function(){

        var notification = new Notification(BACKUP_RUNNING);
        $("#backup_data").before(notification.$el);
        notification.show();

        resourcesService.create_backup(function(data){
            setTimeout(function(){
                getBackupStatus(notification);
            }, 1);
        });
    };

    $scope.restoreBackup = function(filename){

        var notification = new Notification(RESTORE_RUNNING);
        $("#backup_data").before(notification.$el);
        notification.show();

        resourcesService.restore_backup(filename, function(data){
            setTimeout(function(){
                getRestoreStatus(notification);
            }, 1);
        });
    };

    function getBackupStatus(notification){
        resourcesService.get_backup_status(function(data){

            // all done!
            if(data.Detail === ""){
                resourcesService.get_backup_files(function(data){
                    $scope.backupFiles = data;
                });
                notification.successify(BACKUP_COMPLETE);
                return;

            // something neato has happened. lets show it.
            } else if (data.Detail !== "timeout"){
                notification.updateStatus(data.Detail);
            }

            // poll again
            setTimeout(function(){
                getBackupStatus(notification);
            }, 1);

        }, function(data, status){
                notification.failify(ERROR +" "+ status, data.Detail);
        });
    }

    function getRestoreStatus(notification){
        resourcesService.get_restore_status(function(data){

            // all done!
            if(data.Detail === ""){
                notification.successify(RESTORE_COMPLETE);
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
            notification.failify(ERROR +" "+ status, data.Detail);
        });

    }


    // NOTE: this might not be the angular way to do this.
    // plus this notification should eventually be reusable
    // outside of just backup/restore
    
    /**
     * Notification
     * Creates a notification. Great for parties!
     */
    function Notification(title){
        this.$el = $($templateCache.get("backupInfoNotification.html"));
        this.$status = this.$el.find(".backupStatus");
        this.$title = this.$el.find(".backupRunning");
        this.updateTitle(title);

        // bind onClose context so it doesn't have
        // to be rebound for each event listener
        this.onClose = this.onClose.bind(this);
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
            this.$el.find(".close").show().off().on("click", this.onClose);
        },

        // makes notification fail :(
        failify: function(title, msg){
            // change notification color, icon, text, etc
            this.$el.removeClass("bg-info").addClass("bg-danger");
            this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-remove-sign");
            
            this.updateTitle(title);
            this.updateStatus(msg);

            // show close button and make it active
            this.$el.find(".close").show().off().on("click", this.onClose);
        },

        onClose: function(e){
            this.$el.slideUp("fast", function(){
                this.$el.remove();
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
            this.$el.slideDown("fast");
        }
    };
}
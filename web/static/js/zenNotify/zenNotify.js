'use strict';

(function() {

    /**
     * @ngdoc overview
     * @name notification
     */

    angular.module('zenNotify', []).

    /**
     * @ngdoc object
     * @name zenNotify.Notification
     * @requires $templateCache
     */

     factory('$notification', ['$templateCache', '$translate', function ($templateCache, $translate) {
        /**
         * Notification
         * Creates a notification. Great for parties!
         */
        function Notification(id){
            this.id = id;
            this.$el = $($templateCache.get("notification.html"));
            this.$status = this.$el.find(".notification");
            this.$title = this.$el.find(".title");
            this.msg = "";
            this.title = "";

            // bind onClose context so it doesn't have
            // to be rebound for each event listener
            this.onClose = this.onClose.bind(this);

            $("#notifications").append(this.$el);
        }

        Notification.prototype = {
            constructor: Notification,

            success: function(msg){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-success");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-ok-sign");

                this.updateTitle($translate("success"));
                this.updateStatus(msg);

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                return this;
            },

            warning: function(msg){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-warning");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-warning-sign");

                this.updateTitle($translate("warning"));
                this.updateStatus(msg);

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                return this;
            },

            info: function(msg){
                this.updateTitle($translate("info"));
                this.updateStatus(msg);

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                return this;
            },

            error: function(msg){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-danger");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-remove-sign");

                this.updateTitle($translate("error"));
                this.updateStatus(msg);

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                return this;
            },

            onClose: function(e){
                this.$el.slideUp("fast", function(){
                    this.$el.remove();
                }.bind(this));

                NotificationFactory.markRead(this);
            },

            // updates the status message (the smaller text)
            updateStatus: function(msg){
                this.msg = msg || "";
                this.$status.html(this.msg);
                return this;
            },

            // updates the notification title (larger text)
            updateTitle: function(title){
                this.title = title || "";
                this.$title.text(this.title);
                return this;
            },

            show: function(){
                this.$el.slideDown("fast");
                NotificationFactory.store(this);
                return this;
            }
        }

        var NotificationFactory = {
            $storage: JSON.parse(localStorage.getItem('messages')) || [],
            lastId: null,

            create: function(){
                // if this is the first time we sending a message, try to lookup the next ID from storage
                if(this.lastId === null){
                    this.lastId = 0;
                    this.$storage.forEach(function(el, idx){
                        if(el.id >= this.lastId){
                            this.lastId = el.id;
                        }
                    }.bind(this));
                }

                return new Notification(++this.lastId);
            },

            // TODO: Rewrite this as an event listener and add emit to Notification.onClose()
            markRead: function(notification){
                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el.read = true;
                        this.$storage[idx] = el;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
            },

            store: function(notification){
                var storable = {id: notification.id, read: false, title: notification.title, msg: notification.msg}

                if(this.$storage.unshift(storable) > 10){
                    this.$storage.pop();
                }

                localStorage.setItem('messages', JSON.stringify(this.$storage));
            },

            update: function(notification){
                var storable = {id: notification.id, read: false, title: notification.title, msg: notification.msg}

                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el.read = true;
                        this.$storage[idx] = el;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
            }
        };

        return NotificationFactory;
    }]);
})();

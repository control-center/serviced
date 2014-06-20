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

     factory('$notification', ['$rootScope', '$templateCache', '$translate', function ($rootScope, $templateCache, $translate) {
        /**
         * Notification
         * Creates a notification. Great for parties!
         */
        function Notification(id, title, msg, attachPoint){
            this.id = id;
            this.$el = $($templateCache.get("notification.html"));
            this.$status = this.$el.find(".notification");
            this.$title = this.$el.find(".title");
            this.title = title;
            this.msg = msg;
            this.attachPoint = attachPoint;

            // bind onClose context so it doesn't have
            // to be rebound for each event listener
            this.onClose = this.onClose.bind(this);
            this.hide = this.hide.bind(this);
        }

        Notification.prototype = {
            constructor: Notification,

            success: function(){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-success");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-ok-sign");

                this.updateTitle(this.title || $translate("success"));
                this.updateStatus(this.msg || "");

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                NotificationFactory.store(this);
                this.show();
                return this;
            },

            warning: function(){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-warning");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-warning-sign");

                this.updateTitle(this.title || $translate("warning"));
                this.updateStatus(this.msg || "");

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                NotificationFactory.store(this);
                this.show();
                return this;
            },

            info: function(){
                this.updateTitle(this.title || $translate("info"));
                this.updateStatus(this.msg || "");

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                NotificationFactory.store(this);
                this.show();
                return this;
            },

            error: function(){
                // change notification color, icon, text, etc
                this.$el.removeClass("bg-info").addClass("bg-danger");
                this.$el.find(".dialogIcon").removeClass("glyphicon-info-sign").addClass("glyphicon-remove-sign");

                this.updateTitle(this.title || $translate("error"));
                this.updateStatus(this.msg || "");

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                NotificationFactory.store(this);
                this.show();
                return this;
            },

            onClose: function(e){
                NotificationFactory.markRead(this);
                this.hide();
            },

            hide: function(){
                this.$el.slideUp("fast", function(){
                    this.$el.remove();
                }.bind(this));
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

            show: function(autoclose){
                this.attachPoint.append(this.$el);

                autoclose = typeof autoclose !== 'undefined' ? autoclose : true;
                this.$el.slideDown("fast");

                if(autoclose){
                    setTimeout(this.hide(), 5000);
                }

                return this;
            }
        }

        var NotificationFactory = {
            $storage: JSON.parse(localStorage.getItem('messages')) || [],
            lastId: null,

            create: function(title, msg){
                // if this is the first time we sending a message, try to lookup the next ID from storage
                if(this.lastId === null){
                    this.lastId = 0;
                    this.$storage.forEach(function(el, idx){
                        if(el.id >= this.lastId){
                            this.lastId = el.id;
                        }
                    }.bind(this));
                }

                var notification = new Notification(++this.lastId, title, msg, $("#notifications"));
                return notification;
            },

            // TODO: Rewrite this as an event listener and add emit to Notification.onClose()
            markRead: function(notification){
                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el.read = true;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            store: function(notification){
                var storable = {id: notification.id, read: false, date: new Date(), title: notification.title, msg: notification.msg}

                if(this.$storage.unshift(storable) > 10){
                    this.$storage.pop();
                }

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            update: function(notification){
                var storable = {id: notification.id, read: false, title: notification.title, msg: notification.msg}

                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el.read = true;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            getMessages: function(){
                var unreadCount = 0;

                this.$storage.forEach(function(el, idx){
                    if(!el.read){
                        ++unreadCount;
                    }
                });

                return {
                    unreadCount: unreadCount,
                    messages: this.$storage
                };
            },

            clearAll: function(){
                this.$storage = [];
                localStorage.clear();
                $rootScope.$broadcast("messageUpdate");
            }
        };

        return NotificationFactory;
    }]);
})();

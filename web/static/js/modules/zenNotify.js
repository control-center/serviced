/* global: $ */
/* jshint multistr: true */

(function() {
    'use strict';

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
        var notificationFactory;

        var notificationTemplate = '<div class="bg-info notification" style="display:none;">\
            <span class="dialogIcon glyphicon glyphicon-info-sign"></span>\
            <span class="title"></span>\
            <span class="message"></span>\
            <button type="button" class="close" aria-hidden="true" style="display:none;">&times;</button>\
        </div>';

        /**
         * Notification
         * Creates a notification. Great for parties!
         */
        function Notification(id, title, msg, $attachPoint){
            this.id = id;
            this.$el = $(notificationTemplate);
            this.$message = this.$el.find(".message");
            this.$title = this.$el.find(".title");
            this.title = title;
            this.msg = msg;
            this.$attachPoint = $attachPoint;

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
                notificationFactory.store(this);
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
                notificationFactory.store(this);
                this.show();
                return this;
            },

            info: function(){
                this.updateTitle(this.title || $translate("info"));
                this.updateStatus(this.msg || "");

                // show close button and make it active
                this.$el.find(".close").show().off().on("click", this.onClose);
                notificationFactory.store(this);
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
                notificationFactory.store(this);
                this.show(false);
                return this;
            },

            onClose: function(e){
                notificationFactory.markRead(this);
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
                this.$message.html(this.msg);
                return this;
            },

            // updates the notification title (larger text)
            updateTitle: function(title){
                this.title = title || "";
                this.$title.text(this.title);
                return this;
            },

            show: function(autoclose){
                this.$attachPoint.append(this.$el);

                autoclose = typeof autoclose !== 'undefined' ? autoclose : true;
                this.$el.slideDown("fast");

                if(autoclose){
                    setTimeout(this.hide, 5000);
                }

                return this;
            }
        };


        function NotificationFactory(){
            this.$storage = JSON.parse(localStorage.getItem('messages')) || [];
            this.lastId = null;

            // if this is the first time we sending a message, try to lookup the next ID from storage
            if(this.lastId === null){
                this.lastId = 0;
                this.$storage.forEach(function(el, idx){
                    if(el.id >= this.lastId){
                        this.lastId = el.id;
                    }
                }.bind(this));
            }
        }

        /**
         * Notification Factory
         * interface for creating, storing, and updating notifications
         */
        NotificationFactory.prototype = {
            constructor: NotificationFactory,

            /**
             * create a new notification. Loads of fun!
             * @param  {string} title  notification title. treated as plain text
             * @param  {string} msg  notification message. treated as HTML
             * @param  {jQueryObject} $attachPoint  jQuery DOM element to attach notification to
             *                                      defaults to `#notification` element
             * @return {Notification}  returns the Notification object
             */
            create: function(title, msg, $attachPoint){
                // if no valid attachPoint is provided, default to #notifications
                if(!$attachPoint || !$attachPoint.length){
                    $attachPoint = $("#notifications");
                }
                var notification = new Notification(++this.lastId, title, msg, $attachPoint);

                return notification;
            },

            /**
             * marks provided notification read and updates local data store
             * @param  {Notification} notification  the Notification object to mark read
             */
            markRead: function(notification){
                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el.read = true;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            /**
             * stores provided notification
             * @param  {Notification} notification  the Notification object to store
             */
            store: function(notification){
                var storable = {id: notification.id, read: false, date: new Date(), title: notification.title, msg: notification.msg};

                if(this.$storage.unshift(storable) > 10){
                    this.$storage.pop();
                }

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            /**
             * updates stored notification (by id) with the provided notification
             * @param  {Notification} notification  the Notification object to update
             */
            update: function(notification){
                var storable = {id: notification.id, read: false, date: new Date(), title: notification.title, msg: notification.msg};

                this.$storage.forEach(function(el, idx){
                    if(el.id === notification.id){
                        el = storable;
                    }
                }.bind(this));

                localStorage.setItem('messages', JSON.stringify(this.$storage));
                $rootScope.$broadcast("messageUpdate");
            },

            /**
             * gets all stored messages as well as number of unread messages
             * @return {object}  object containing `unreadCount` - the number of unread messages,
             *                          and `messages` - an array of stored notifications.
             */
            getMessages: function(){
                var unreadCount;

                unreadCount = this.$storage.reduce(function(acc, el){
                    return !el.read ? acc+1 : acc;
                }, 0);

                return {
                    unreadCount: unreadCount,
                    messages: this.$storage
                };
            },

            /**
             * removes all stored Notifications (read and unread)
             */
            clearAll: function(){
                this.$storage = [];
                localStorage.clear();
                $rootScope.$broadcast("messageUpdate");
            }
        };

        notificationFactory = new NotificationFactory();
        return notificationFactory;
    }]);
})();

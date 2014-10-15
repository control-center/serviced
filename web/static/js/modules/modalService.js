/* global angular, console, $ */
/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('modalService', []).
    factory("$modalService", [
        "$rootScope", "$templateCache", "$http", "$interpolate", "$compile", "$translate",
        function($rootScope, $templateCache, $http, $interpolate, $compile, $translate){

            var defaultModalTemplate = '<div class="modal fade" tabindex="-1" role="dialog" aria-hidden="true">\
                <div class="modal-dialog {{bigModal}}">\
                    <div class="modal-content">\
                        <div class="modal-header">\
                            <button type="button" class="close glyphicon glyphicon-remove-circle" data-dismiss="modal" aria-hidden="true"></button>\
                            <span class="modal-title">{{title}}</span>\
                        </div>\
                        <div class="modal-body">{{template}}</div>\
                        <div class="modal-footer"></div>\
                    </div>\
                </div>>\
            </div>';

            var actionButtonTemplate = '<button type="button" class="btn {{classes}}">{{label}}</button>';

            var defaultRoles = {
                "cancel": {
                    label: "Cancel",
                    icon: "glyphicon-close",
                    classes: "btn-link",
                    action: function(){
                        this.close();
                    }
                },
                "ok": {
                    label: "Ok",
                    icon: "glyphicon-ok",
                    classes: "btn-primary",
                    action: function(){
                        this.close();
                    }
                }
            };

            /**
             * Modal window
             */
            function Modal(template, model, config){
                var $modalFooter;
                // inject user provided template into modal template
                var modalTemplate = $interpolate(defaultModalTemplate)({
                    template: template,
                    title: $translate.instant(config.title),
                    bigModal: config.bigModal ? "big" : ""
                });

                // bind user provided model to final modal template
                this.$el = $($compile(modalTemplate)(model)).modal();

                $modalFooter = this.$el.find(".modal-footer");

                // create action buttons
                config.actions.forEach(function(action){

                    // if this action has a role on it, merge role defaults
                    if(action.role && defaultRoles[action.role]){
                        for(var i in defaultRoles[action.role]){
                            action[i] = action[i] || defaultRoles[action.role][i];
                        }
                    }

                    // translate button label
                    action.label = $translate.instant(action.label);

                    var $button = $($interpolate(actionButtonTemplate)(action));
                    $button.on("click", action.action.bind(this));
                    $modalFooter.append($button);
                }.bind(this));

                // if no actions, remove footer
                if(!config.actions.length){
                    $modalFooter.remove();
                }

                // setup/default validation function
                this.validateFn = config.validate || function(){ return true; };

                // listen for hide event and completely remove modal
                // after it is hidden
                this.$el.on("hidden.bs.modal", function(){
                    this.destroy();
                }.bind(this));
            }

            Modal.prototype = {
                constructor: Modal,
                close: function(){
                    this.$el.modal("hide");
                    
                },
                show: function(){
                    this.$el.modal("show");
                },
                validate: function(){
                    return this.validateFn();
                },
                destroy: function(){
                    this.$el.remove();
                }
            };
            



            var modalsPath = "/static/partials/",
                // keep track of existing modals so that they can be
                // close/destroyed when a new one is created
                // TODO - remove modals from this list when they are hidden
                modals = [];

            /**
             * Fetches modal template and caches it in $templateCache.
             * returns a promise for the http request
             */
            function fetchModalTemplate(name){
                var url = modalsPath + name;
                return $http.get(url, {cache: $templateCache});
            }

            /**
             * fetches modal template and passes it along to be attached
             * to the DOM
             */
            function create(config){

                config = config || {};
                // TODO - default config object
                config.actions = config.actions || [];
                config.onShow = config.onShow || function(){};
                
                var model = config.model || {};

                // if the template was provided, use that
                if(config.template){
                    _create(config.template, model, config);

                // otherwise, request the template
                // TODO - pop a modal with load spinner?
                } else {
                    fetchModalTemplate(config.templateUrl).then(function(res){
                        _create(res.data, model, config);
                    });
                }
            }

            function _create(template, model, config){
                var modal = new Modal(template, model, config);
                modal.show();

                // immediately destroy any existing modals
                modals.forEach(function(momo){
                    momo.destroy();
                });
                modals = [modal];

                // perform onShow function after modal is visible
                modal.$el.one("shown.bs.modal.", config.onShow.bind(modal));
            }

            return {
                create: create
            };

        }
    ]);
})();
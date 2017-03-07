/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('modalService', []).
    factory("$modalService", [
        "$rootScope", "$templateCache", "$http", "$interpolate", "$compile", "$translate", "$notification",
        "miscUtils", "CCUIState",
        function($rootScope, $templateCache, $http, $interpolate, $compile, $translate, $notification,
        utils, CCUIState){

            // accessing certain properties forces a DOM reflow,
            // which is useful if you want some CSS changes to
            // be applied to force the next CSS change to trigger
            // a transition
            function pokeDOM(){
                return document.body.scrollTop;
            }

            // global, reusable, handy-dandy modal
            // backdrop element, guaranteed to reduce all
            // other content down to about 50% of its original
            // brightness or your money back!
            class ModalDarkener{
                constructor(){
                    this.el = document.createElement("div");
                    this.el.className = "modal-darkener";
                    document.body.appendChild(this.el);
                }

                show(){
                    this.el.style.display = "block";
                    pokeDOM();
                    this.el.classList.add("show");
                }

                hide(){
                    this.el.classList.remove("show");
                    // TODO - css transition end
                    setTimeout(() => {
                        this.el.style.display = "none";
                    }, 250);
                }
            }
            var darkener = new ModalDarkener();

            var defaultModalTemplate = function(model){
                return `
                    <div class="modal fade" tabindex="-1" role="dialog" aria-hidden="true">
                        <div class="modal-dialog ${model.bigModal}">
                            <div class="modal-content">
                                <div class="modal-header">
                                    ${model.unclosable ?
                                        `` :
                                        `<button type="button" class="close glyphicon glyphicon-remove-circle" data-dismiss="modal" aria-hidden="true"></button>`}
                                    <span class="modal-title">${model.title}</span>
                                </div>
                                <div class="modal-notify"></div>
                                <div class="modal-body">${model.template}</div>
                                <div class="modal-footer"></div>
                            </div>
                        </div>
                    </div>`;
            };

            var actionButtonTemplate = '<button type="button" class="btn {{classes}}"><span ng-show="icon" class="glyphicon {{icon}}"></span> {{label}}</button>';

            var defaultRoles = {
                "cancel": {
                    label: "btn_cancel",
                    icon: "glyphicon-remove",
                    classes: "btn-link minor",
                    action: function(){
                        this.close();
                    }
                },
                "ok": {
                    label: "Ok",
                    icon: "glyphicon-ok",
                    classes: "btn-primary submit",
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
                var modalTemplate = defaultModalTemplate({
                    template: template,
                    title: $translate.instant(config.title),
                    bigModal: config.bigModal ? "big" : "",
                    unclosable: config.unclosable
                });

                let bootstrapModalConfig = {
                    backdrop: false,
                    keyboard: !config.unclosable
                };

                // bind user provided model to final modal template
                this.$el = $($compile(modalTemplate)(model)).modal(bootstrapModalConfig);

                // enforce disabling animation on modals if necessary
                if(CCUIState.get().disableAnimation){
                    console.log("disabling animation");
                    this.$el.removeClass("fade");
                }

                $modalFooter = this.$el.find(".modal-footer");
                // cache a reference to the notification holder
                this.$notificationEl = this.$el.find(".modal-notify");

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
                this.validateFn = config.validate || function(args){ return true; };

                // listen for hide event and completely remove modal
                // after it is hidden
                this.$el.on("hidden.bs.modal", function(){
                    darkener.hide();
                    this.destroy();
                }.bind(this));

                // NOTE - internal boostrap modal event that we need
                // to hook into to hide the modal if the darkener is clicked
                this.$el.on("click.dismiss.modal", (e) => {
                    // if clicking the backdrop or clicking an element
                    // marked with class "close", close things
                    if(e.target === e.currentTarget || e.target.classList.contains("close")){
                        this.destroy();
                        darkener.hide();
                    }
                });
            }

            Modal.prototype = {
                constructor: Modal,
                close: function(){
                    this.$el.modal("hide");
                    darkener.hide();
                },
                show: function(){
                    this.$el.modal("show");
                    this.disableScroll();
                    darkener.show();
                },
                validate: function(args){
                    return this.validateFn(args);
                },
                destroy: function(){
                    this.$el.remove();
                    this.enableScroll();
                },
                // convenience method for attaching notifications to the modal
                createNotification: function(title, message){
                    return $notification.create(title, message, this.$notificationEl);
                },

                disableAction(classname) {
                    this.$el.find(".modal-footer button." + classname ).addClass("disabled");
                },

                enableAction(classname) {
                    this.$el.find(".modal-footer button." + classname ).removeClass("disabled");
                },

                disableScroll(){
                    var bodyEl = $("body");
                    bodyEl.css("overflow", "hidden");
                },
                enableScroll(){
                    $("body").css("overflow", "scroll");
                },

                // convenience method to disable the default ok/submit button
                disableSubmitButton: function(selector, disabledText){
                    selector = selector || ".submit";
                    disabledText = disabledText || "Submitting...";

                    var $button = this.$el.find(selector),
                        $buttonClone,
                        buttonContent, startWidth, endWidth;

                    // button wasnt found
                    if(!$button.length){
                        return;
                    }

                    // explicitly set width so it can be animated
                    startWidth = $button.width();

                    // clone the button and set the ending text so the
                    // explicit width can be calculated
                    $buttonClone = $button.clone().width("auto").text(disabledText).appendTo("body");
                    endWidth = $buttonClone.width();
                    $buttonClone.remove();

                    $button.width(startWidth);

                    buttonContent = $button.html();
                    $button.prop("disabled", true)
                        .addClass("disabled")
                        .text(disabledText)
                        .width(endWidth);

                    // return a function used to reenable the button
                    return function(){
                        $button.prop("disabled", false)
                            .removeClass("disabled")
                            .html(buttonContent)
                            .width(startWidth);
                    };
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
                config.onHide = config.onHide || function(){};
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
                // immediately destroy any existing modals
                modals.forEach(function(momo){
                    momo.destroy();
                });

                var modal = new Modal(template, model, config);
                modal.show();

                modals = [modal];

                // perform onShow function after modal is visible
                modal.$el.one("shown.bs.modal.", function(){
                    // search for and autofocus the focusme element
                    // unless disableAnimation is set, which probably
                    // indicates this is an acceptance test is horrendously
                    // slow and prone to breaking
                    if(!CCUIState.get().disableAnimation){
                        modal.$el.find("[focusme]").first().focus();
                    }

                    // call user provided onShow function
                    setTimeout(() => {
                        config.onShow.call(modal);
                    }, 0);
                });

                modal.$el.one("hidden.bs.modal.", config.onHide.bind(modal));

            }

            let displayHostKeys = function(keys, registered, name) {
                let model = $rootScope.$new(true);
                model.keys = keys;
                model.name = name;
                model.registered = registered;

                create({
                    templateUrl: "display-host-keys.html",
                    model: model,
                    title: $translate.instant("title_host_keys"),
                    actions: [
                        {
                            label: $translate.instant("btn_download_keys"),
                            action: function(){
                                utils.downloadText(name + ".keys", keys);
                            },
                            icon: "glyphicon-download"
                        },{
                            role: "ok"
                        }
                    ],
                    onShow: function(){
                        // TODO - dont touch the DOM!
                        let keysWrapEl = this.$el.find(".keys-wrap"),
                            keysEl = keysWrapEl.find(".keys");
                        if (model.registered) {
                            this.createNotification("", "Host keys registered automatically").success();
                        }
                        keysWrapEl.on("click", e => {
                            // TODO - if already selected, this deselects
                            keysEl.select();
                            try {
                                let success = document.execCommand('copy');
                                if(success){
                                    this.createNotification("", "Keys copied to clipboard").info();
                                } else {
                                    this.createNotification("", "Press Ctrl+C or Cmd+C to copy keys").info();
                                }
                            } catch(err) {
                                this.createNotification("", "Press Ctrl+C or Cmd+C to copy keys").info();
                            }
                        });
                    }
                });
            };

            let oneMoment = function(message) {
                let model = $rootScope.$new(true);
                model.message = message || $translate.instant("one_moment");
                let html = `
                    <div style="width: 100%; text-align: center;">
                        <img src="static/img/loading.gif">
                        <div style="max-width: 75%; margin: 10px auto;">${model.message}</div>
                    </div>`;

                create({
                    template: html,
                    model: model,
                    title: $translate.instant("one_moment"),
                    unclosable: true
                });
            };

            let confirmServiceStateChange = function(service, state, childCount, onStartService, onStartServiceAndChildren){
                let manyTemplate = function(model){
                    return `
                        <h4>${$translate.instant("choose_services_" + model.state)}</h4>
                        <ul>
                            <li>${$translate.instant(state + "_service_name", { name: "<strong>" + model.service.name + "</strong>" })}</li>
                            <li>${$translate.instant(state + "_service_name_and_children", {
                                 name: `<strong>${model.service.name}</strong>`,
                                 count: `<strong>${model.childCount}</strong>` }
                            )}</li>
                        </ul>`;
                };

                let singleTemplate = function(model){
                    return $translate.instant(
                        "service_will_" + model.state,
                        {name: `<strong>${model.service.name}</strong>`});
                };
                
                let model = $rootScope.$new(true);
                model = angular.extend(model, {service, state, childCount});
                let html;
                // button actions for the modal
                let actions = [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        classes: " ",
                        label: $translate.instant(state + "_service"),
                        action: function () {
                            onStartService(this);
                        }
                    }
                ];

                // If there are any child nodes affected (1+ children), give them the
                // option to just start the service or service + 1 child.
                if(childCount >= 1) {
                    html = manyTemplate(model);
                    actions.push({
                        role: "ok",
                        label: $translate.instant(state + "_service_and_children", { count: childCount }),
                        action: function () {
                            onStartServiceAndChildren(this);
                        }
                    });
                } else {
                     html = singleTemplate(model);
                }

                create({
                    template: html,
                    model: model,
                    title: $translate.instant(state + "_service"),
                    actions: actions
                });
            };

            return {
                create: create,
                // some shared modals that anyone can enjoy!
                modals: {
                    displayHostKeys,
                    oneMoment,
                    confirmServiceStateChange
                }
            };

        }
    ]);
})();

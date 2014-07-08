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
					title: $translate(config.title),
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
					action.label = $translate(action.label);

					var $butt = $($interpolate(actionButtonTemplate)(action));
					$butt.on("click", action.action.bind(this));
					$modalFooter.append($butt);
				}.bind(this));

				// setup/default validation function
				this.validateFn = config.validate || function(){};

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
				var url = modalsPath + name + ".html";
				return $http.get(url, {cache: $templateCache});
			}

			/**
			 * creates a modal and attaches to the DOM
			 * @param  {string} templateName  name of the template to use for the modal
			 *                                it is fetched from a url defined in modalService
			 * @param  {object} model  model to bind to template
			 */
			function createModal(templateName, model, config){

				config = config || {};
				
				model = model || {};

				fetchModalTemplate(templateName).then(function(res){
					var modal = new Modal(res.data, model, config);
					modal.show();

					// immediately destroy any existing modals
					modals.forEach(function(momo){
						momo.destroy();
					});
					modals = [modal];
				});
			}

			return {
				createModal: createModal
			};

		}
	]);
})();
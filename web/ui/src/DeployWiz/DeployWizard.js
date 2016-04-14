/* globals controlplane: true */

/* DeployWizard.js
 * Guides user through deployment of an app
 */
(function() {
    'use strict';

    controlplane.controller("DeployWizard", ["$scope", "$notification", "$translate", "$q", "resourcesFactory", "servicesFactory", "miscUtils", "hostsFactory", "poolsFactory",
    function($scope, $notification, $translate, $q, resourcesFactory, servicesFactory, utils, hostsFactory, poolsFactory){
        var step = 0;
        var nextClicked = false;
        $scope.name='wizard';

        $scope.dockerLoggedIn = true;

        resourcesFactory.dockerIsLoggedIn()
            .success(function(loggedIn) {
                $scope.dockerLoggedIn = loggedIn;
            });

        $scope.dockerIsNotLoggedIn = function() {
            return !$scope.dockerLoggedIn;
        };

        var validTemplateSelected = function() {
            if(!$scope.install.templateID){
                showError($translate.instant("label_wizard_select_app"));
                return false;
            }else{
                resetError();
            }

            return true;
        };

        var validDeploymentID = function() {
            if($scope.install.deploymentID === undefined || $scope.install.deploymentID === ""){
                showError($translate.instant("label_wizard_deployment_id"));
                return false;
            }else{
                resetError();
            }

            return true;
        };

        var validTemplateUpload = function(){
            var uploadedFiles = $("#new_template_filename_wizard")[0].files;
            if(uploadedFiles.length === 0){
                showError($translate.instant("template_error"));
                return false;
            }else{
                var formData = new FormData();
                $.each(uploadedFiles, function(key, value){
                    formData.append("tpl", value);
                });
                resourcesFactory.addAppTemplate(formData)
                    .success($scope.refreshAppTemplates)
                    .error(() => {
                        showError("Add Application Template failed");
                    });

                resetError();
                return true;
            }
        };

        var validHost = function(){
            var err = utils.validateHostName($scope.newHost.host, $translate) ||
                utils.validatePortNumber($scope.newHost.port, $translate) ||
                utils.validateRAMLimit($scope.newHost.RAMLimit);
            if(err){
                showError(err);
                return false;
            }

            $scope.newHost.IPAddr = $scope.newHost.host + ':' + $scope.newHost.port;

            resourcesFactory.addHost($scope.newHost)
                .success(function(){
                    step += 1;
                    resetError();
                    $scope.step_page = $scope.steps[step].content;
                })
                .error(function(data){
                    // if it already exists then allow the user to continue
                    if (data.Detail.indexOf('already exists') !== -1) {
                        step += 1;
                        resetError();
                        $scope.step_page = $scope.steps[step].content;
                    } else {
                        showError("Add Host failed", data.Detail);
                    }
                });




            return false;
        };

        var resetStepPage = function() {
            step = 0;

            $scope.install = {
                poolID: 'default'
            };

            if($scope.templates.data.length === 0){
                $scope.steps.unshift({
                    content: '/static/partials/wizard-modal-add-template.html',
                    label: 'template_add',
                    validate: validTemplateUpload
                });
            }

            // if there is not at least one host, add an
            // "add host" step to the wizard
            if(hostsFactory.hostList.length === 0){
                $scope.newHost = {
                    port: $translate.instant('placeholder_port')
                };
                if ($scope.pools.length > 0){
                    $scope.newHost.PoolID = $scope.pools[0].id;
                }
                $scope.steps.unshift({
                    content: '/static/partials/wizard-modal-add-host.html',
                    label: 'add_host',
                    validate: validHost
                });
            }

            $scope.step_page = $scope.steps[step].content;
        };

        var showError = function(message){
            $("#deployWizardNotificationsContent").html(message);
            $("#deployWizardNotifications").removeClass("hide");
        };

        var resetError = function(){
            $("#deployWizardNotifications").html("");
            $("#deployWizardNotifications").addClass("hide");
        };

        $scope.steps = [
            {
                content: '/static/partials/wizard-modal-2.html',
                label: 'label_step_select_app',
                validate: validTemplateSelected
            },
            {
                content: '/static/partials/wizard-modal-3.html',
                label: 'label_step_select_pool' },
            {
                content: '/static/partials/wizard-modal-4.html',
                label: 'label_step_deploy',
                validate: validDeploymentID
            }
        ];

        $scope.install = {
            poolID: 'default'
        };

        $scope.selectTemplate = function(template){
            $scope.template = template;
            $scope.install.templateID = template.ID;
        };

        $scope.selectPool = function(pool){
            $scope.install.poolID = pool.id;
        };

        $scope.getTemplateRequiredResources = function(template){
            var ret = {CPUCommitment:0, RAMCommitment:0};

            // if Services, iterate and sum up their commitment values
            if(template.Services){
                var suffixToMultiplier = {
                    "":  1,
                    "k": 1 << 10,
                    "m": 1 << 20,
                    "g": 1 << 30,
                    "t": 1 << 40
                };
                var engNotationRE = /([0-9]*)([kKmMgGtT]?)/;
                // Convert an engineeringNotation string to a number
                var toBytes = function(RAMCommitment){
                    if (RAMCommitment === "") {
                        return 0;
                    }
                    var match = RAMCommitment.match(engNotationRE);
                    var numeric = match[1];
                    var suffix = match[2].toLowerCase();
                    var multiplier = suffixToMultiplier[suffix];
                    var val = parseInt(numeric);
                    return val * multiplier;
                };
                // recursively calculate cpu and ram commitments
                (function calcCommitment(services){
                    services.forEach(function(service){
                        // CPUCommitment should be equal to max number of
                        // cores needed by any service
                        ret.CPUCommitment = Math.max(ret.CPUCommitment, service.CPUCommitment);
                        // RAMCommitment should be a sum of all ram needed
                        // by all services
                        ret.RAMCommitment += toBytes(service.RAMCommitment);

                        // recurse!
                        if(service.Services){
                            calcCommitment(service.Services);
                        }
                    });
                })(template.Services);
            }

            return ret;
        };

        $scope.addHostStart = function() {
            $scope.newHost = {
                port: $translate.instant('placeholder_port')
            };
            if ($scope.pools.length > 0){
                $scope.newHost.PoolID = $scope.pools[0].id;
            }
            $scope.step_page = '/static/partials/wizard-modal-addhost.html';
        };

        $scope.hasPrevious = function() {
            return step > 0 &&
                ($scope.step_page === $scope.steps[step].content);
        };

        $scope.hasNext = function() {
            return (step + 1) < $scope.steps.length &&
                ($scope.step_page === $scope.steps[step].content);
        };

        $scope.hasFinish = function() {
            return (step + 1) === $scope.steps.length;
        };

        $scope.step_item = function(index) {
            var cls = index <= step ? 'active' : 'inactive';
            if (index === step) {
                cls += ' current';
            }
            return cls;
        };

        $scope.step_label = function(index) {
            return index < step ? 'done' : '';
        };

        $scope.wizard_next = function() {
            nextClicked = true;

            if ($scope.step_page !== $scope.steps[step].content) {
                $scope.step_page = $scope.steps[step].content;
                nextClicked = false;
                return;
            }

            if ($scope.steps[step].validate) {
                if (!$scope.steps[step].validate()) {
                    nextClicked = false;
                    return;
                }
            }

            step += 1;
            resetError();
            $scope.step_page = $scope.steps[step].content;
            nextClicked = false;
        };

        $scope.wizard_previous = function() {
            step -= 1;
            $scope.step_page = $scope.steps[step].content;
            resetError();
        };

        $scope.wizard_finish = function() {

            var closeModal = function(){
                $('#addApp').modal('hide');
                $("#deploy-save-button").removeAttr("disabled");
                $("#deploy-save-button").removeClass('active');
                resetStepPage();
                resetError();
            };

            nextClicked = true;
            if ($scope.steps[step].validate) {
                if (!$scope.steps[step].validate()) {
                    return;
                }
            }

            $("#deploy-save-button").toggleClass('active');
            $("#deploy-save-button").attr("disabled", "disabled");

            var deploymentDefinition = {
                poolID: $scope.install.poolID,
                TemplateID: $scope.install.templateID,
                DeploymentID: $scope.install.deploymentID
            };

            var checkStatus = true;
            resourcesFactory.deployAppTemplate(deploymentDefinition)
                .success(function() {
                    checkStatus = false;
                    servicesFactory.update(true, false);
                    $notification.create("App deployed successfully").success();
                    closeModal();
                })
                .error(function(data, status){
                    checkStatus = false;
                    $notification.create("App deploy failed", data.Detail).error();
                    closeModal();
                });

            //now that we have started deploying our app, we poll for status
            var getStatus = function(){
                if(checkStatus){
                    var $status = $("#deployStatusText");
                    resourcesFactory.getDeployStatus(deploymentDefinition)
                        .success(function(data){
                            if(data.Detail === "timeout"){
                                $("#deployStatus .dialogIcon").fadeOut(200, function(){$("#deployStatus .dialogIcon").fadeIn(200);});
                            }else{
                                var parts = data.Detail.split("|");
                                if(parts[1]){
                                    $status.html('<strong>' + $translate.instant(parts[0]) + ":</strong> " + parts[1]);
                                }else{
                                    $status.html('<strong>' + $translate.instant(parts[0]) + '</strong>');
                                }
                            }
                        })
                        .error(function(err){
                            console.warn("Error getting deploy status", err);
                        })
                        .finally(function(){
                            getStatus();
                        });
                }
            };

            $("#deployStatus").show();
            getStatus();

            nextClicked = false;
        };

        $scope.refreshAppTemplates = function(){
            const MAX_RETRIES = 4;
            var deferred = $q.defer(),
                attempts = 0;

            // allow requests to be repeated if necessary
            var fetch = () => {
                resourcesFactory.getAppTemplates()
                    .then(function(templatesMap) {
                        var templates = [];
                        for (var key in templatesMap) {
                            var template = templatesMap[key];
                            template.ID = key;
                            templates.push(template);
                        }
                        $scope.templates.data = templates;
                        deferred.resolve();
                    }, function(){
                        if(attempts >= MAX_RETRIES){
                            deferred.reject("Unable to refresh application templates");
                        }
                        // retry in 3s
                        setTimeout(fetch, 3000);
                        attempts++;
                    });
            };
            fetch();

            return deferred.promise;
        };

        $scope.refreshAppTemplates()
            .then(() => {
                hostsFactory.update().finally(resetStepPage);
            }, e => {
                console.error(e);
            });

        poolsFactory.update()
            .finally(() => {
                $scope.pools = poolsFactory.poolList;
            });
    }]);
})();

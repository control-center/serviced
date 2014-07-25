function DeployWizard($scope, $notification, $translate, $http, resourcesService) {
    var step = 0;
    var nextClicked = false;
    $scope.name='wizard';

    var  validTemplateSelected = function() {
        if($scope.selectedTemplates().length <= 0){
            showError($translate("label_wizard_select_app"));
            return false;
        }

        return true;
    };

    var validDeploymentID = function() {
        if($scope.install.deploymentId === undefined || $scope.install.deploymentId === ""){
            showError($translate("label_wizard_deployment_id"));
            return false;
        }

        return true;
    };

    var resetStepPage = function() {
        step = 0;
        $scope.step_page = $scope.steps[step].content;

        $scope.install = {
            selected: {
                pool: 'default'
            },
            templateSelected: function(template) {
                if (template.depends) {
                    $scope.install.selected[template.depends] = true;
                }
            },
            templateDisabled: function(template) {
                if (template.disabledBy) {
                    return $scope.install.selected[template.disabledBy];
                }
                return false;
            }
        };

        resourcesService.get_app_templates(false, function(templatesMap) {
            var templates = [];
            for (var key in templatesMap) {
                var template = templatesMap[key];
                template.Id = key;
                templates[templates.length] = template;
            }
            $scope.install.templateData = templates;
        });
    };

    var showError = function(message){
        $("#deployWizardNotificationsContent").html(message);
        $("#deployWizardNotifications").removeClass("hide");
    }

    var resetError = function(){
        $("#deployWizardNotifications").html("");
        $("#deployWizardNotifications").addClass("hide");
    }

    $scope.steps = [
        /*        { content: '/static/partials/wizard-modal-1.html', label: 'label_step_select_hosts' }, */
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
        selected: {
            pool: 'default'
        },
        templateSelected: function(template) {
            if (template.depends) {
                $scope.install.selected[template.depends] = true;
            }
        },
        templateDisabled: function(template) {
            if (template.disabledBy) {
                return $scope.install.selected[template.disabledBy];
            }
            return false;
        }
    };

    $scope.selectedTemplates = function() {
        var templates = [];
        for (var i=0; i < $scope.install.templateData.length; i++) {
            var template = $scope.install.templateData[i];
            if ($scope.install.selected[template.Id]) {
                templates[templates.length] = template;
            }
        }
        return templates;
    };

    $scope.getTemplateRequiredResources = function(template){
        var ret = {CPUCommitment:0, RAMCommitment:0};
        for (var i=0; i<template.Services.length; ++i){
            if(template.Services[i].CPUCommitment) ret.CPUCommitment += template.Services[i].CPUCommitment;
            if(template.Services[i].RAMCommitment) ret.RAMCommitment += template.Services[i].RAMCommitment;
        }

        return ret;
    }

    $scope.addHostStart = function() {
        $scope.newHost = {};
        $scope.step_page = '/static/partials/wizard-modal-addhost.html';
    };

    $scope.addHostCancel = function() {
        $scope.step_page = $scope.steps[step].content;
    };

    $scope.addHostFinish = function() {
        $scope.newHost.Name = $scope.newHost.IPAddr;
        $scope.newHost.ID = 'fakefakefake';
        $scope.newHost.selected = true;
        $scope.detected_hosts.push($scope.newHost);
        $scope.step_page = $scope.steps[step].content;
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
            return;
        }

        if ($scope.steps[step].validate) {
            if (!$scope.steps[step].validate()) {
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

        closeModal = function(){
            $('#addApp').modal('hide');
            $("#deploy-save-button").removeAttr("disabled");
            $("#deploy-save-button").removeClass('active');
												$("#deploy-start-save-button").removeAttr("disabled");
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
								$("#deploy-start-save-button").attr("disabled", "disabled");

        var selected = $scope.selectedTemplates();
        var f = true;
        var dName = "";
        for (var i=0; i < selected.length; i++) {
            if (f) {
                f = false;
            } else {
                dName += ", ";
                if (i + 1 === selected.length) {
                    dName += "and ";
                }
            }
            dName += selected[i].Name;

            var deploymentDefinition = {
                poolID: $scope.install.selected.pool,
                TemplateID: selected[i].Id,
                DeploymentID: $scope.install.deploymentId
            };

            var checkStatus = true;
            resourcesService.deploy_app_template(deploymentDefinition, function(result) {
                refreshServices($scope, resourcesService, false, function(){
                    //start the service if requested
                    if($scope.install.startNow){
                        for(var i=0; i < $scope.services.data.length; ++i){
                            if (result.Detail == $scope.services.data[i].ID){
                                toggleRunning($scope.services.data[i], "start", resourcesService);
                            }
                        }
                    }
                    checkStatus = false;
                    closeModal();
                });
            }, function(){
                checkStatus = false;
                closeModal();
            });

            //now that we have started deploying our app, we poll for status
            var getStatus = function(){
                if(checkStatus){
                    var $status = $("#deployStatusText");
                    $http.post('/templates/deploy/status', deploymentDefinition).
                        success(function(data, status) {
                            if(data.Detail === "timeout"){
                                $("#deployStatus .dialogIcon").fadeOut(200, function(){$("#deployStatus .dialogIcon").fadeIn(200);});
                            }else{
                                var parts = data.Detail.split("|");
                                if(parts[1]){
                                    $status.html('<strong>' + $translate(parts[0]) + ":</strong> " + parts[1]);
                                }else{
                                    $status.html('<strong>' + $translate(parts[0]) + '</strong>');
                                }
                            }

                            getStatus();
                        });
                }
            };

            $("#deployStatus").show();
            getStatus();
        }

        nextClicked = false;
    };

    $scope.wizard_deploy_start = function(){
        $scope.install.startNow = true;
        $scope.wizard_finish();
    };

    $scope.detected_hosts = [];

    $scope.no_detected_hosts = ($scope.detected_hosts.length < 1);


    resetStepPage();
    refreshPools($scope, resourcesService, true);
}

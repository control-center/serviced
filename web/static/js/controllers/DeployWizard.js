function DeployWizard($scope, resourcesService) {
    $scope.name='wizard';

    var validTemplateSelected = function() {
        return $scope.selectedTemplates().length > 0;
    };

    var validDeploymentID = function() {
        return $scope.install.deploymentId != undefined && $scope.install.deploymentId != "";
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
        },
        templateSelectedFormDiv: function() {
            return (!nextClicked || validTemplateSelected())?
                '':'has-error';
        },
        deploymentIdFormDiv: function() {
            return (!nextClicked || validDeploymentID()) ? '':'has-error';
        }
    };
    var nextClicked = false;

    resourcesService.get_app_templates(false, function(templatesMap) {
        var templates = [];
        for (var key in templatesMap) {
            var template = templatesMap[key];
            template.Id = key;
            templates[templates.length] = template;
        }
        $scope.install.templateData = templates;
    });

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

    var step = 0;
    var resetStepPage = function() {
        step = 0;
        $scope.step_page = $scope.steps[step].content;
    };

    $scope.addHostStart = function() {
        $scope.newHost = {};
        $scope.step_page = '/static/partials/wizard-modal-addhost.html';
    };

    $scope.addHostCancel = function() {
        $scope.step_page = $scope.steps[step].content;
    }

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
        $scope.step_page = $scope.steps[step].content;
        nextClicked = false;
    };

    $scope.wizard_previous = function() {
        step -= 1;
        $scope.step_page = $scope.steps[step].content;
    };

    $scope.wizard_finish = function() {
        $("#deploy-save-button").toggleClass('active');

        nextClicked = true;
        if ($scope.steps[step].validate) {
            if (!$scope.steps[step].validate()) {
                return;
            }
        }

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


            resourcesService.deploy_app_template({
                poolID: $scope.install.selected.pool,
                TemplateID: selected[i].Id,
                DeploymentID: $scope.install.deploymentId
            }, function(result) {
                refreshServices($scope, resourcesService, false, function(){
                    //start the service if requested
                    if($scope.install.startNow){
                        for(var i=0; i < $scope.services.data.length; ++i){
                            if (result.Detail == $scope.services.data[i].Id){
                                toggleRunning($scope.services.data[i], "start", resourcesService);
                            }
                        }
                    }
                });

                $('#addApp').modal('hide');
                resetStepPage();
            }, function(result){
                $('#addApp').modal('hide');
                resetStepPage();
            }
            );
        }

        $scope.services.deployed = {
            name: dName,
            multi: (selected.length > 1),
            class: "deployed alert alert-success",
            show: true,
            deployment: "ready"
        };

        nextClicked = false;
    };

    $scope.detected_hosts = [
        { Name: 'Hostname A', IPAddr: '192.168.34.1', Id: 'A071BF1' },
        { Name: 'Hostname B', IPAddr: '192.168.34.25', Id: 'B770DAD' },
        { Name: 'Hostname C', IPAddr: '192.168.33.99', Id: 'CCD090B' },
        { Name: 'Hostname D', IPAddr: '192.168.33.129', Id: 'DCDD3F0' }
    ];
    $scope.no_detected_hosts = ($scope.detected_hosts.length < 1);

    resetStepPage();

    // Get a list of pools (cached is OK)
    refreshPools($scope, resourcesService, true);
}

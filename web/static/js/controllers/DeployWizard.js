function DeployWizard($scope, $notification, $translate, resourcesService) {
    var step = 0;
    var nextClicked = false;
    $scope.name='wizard';

    $scope.dockerLoggedIn = true;

    resourcesService.docker_is_logged_in(function(loggedIn) {
        $scope.dockerLoggedIn = loggedIn;
    });

    $scope.dockerIsNotLoggedIn = function() {
        return !$scope.dockerLoggedIn;
    };

    var  validTemplateSelected = function() {
        if($scope.selectedTemplates().length <= 0){
            showError($translate.instant("label_wizard_select_app"));
            return false;
        }else{
            resetError();
        }

        return true;
    };

    var validDeploymentID = function() {
        if($scope.install.deploymentId === undefined || $scope.install.deploymentId === ""){
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
            var data = new FormData();
            $.each(uploadedFiles, function(key, value){
                data.append("tpl", value);
            });
            resourcesService.add_app_template(data, function(data){
                resourcesService.get_app_templates(false, function(templatesMap) {
                    var templates = [];
                    for (var key in templatesMap) {
                        var template = templatesMap[key];
                        template.Id = key;
                        templates[templates.length] = template;
                    }
                    $scope.templates.data = templates;
                });
            });
            resetError();
            return true;
        }
    };

    var validHost = function(){
        if($("#new_host_name").val() === ""){
            showError($translate.instant("invalid_host_error"));
            return false;
        }

        resourcesService.add_host($scope.newHost)
            .success(function(){
                step += 1;
                resetError();
                $scope.step_page = $scope.steps[step].content;
            })
            .error(function(data){
                showError(data.Detail);
            });

        return false;
    };

    var resetStepPage = function() {
        step = 0;

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

        if($scope.templates.data.length === 0){
            $scope.steps.unshift({
                content: '/static/partials/wizard-modal-add-template.html',
                label: 'template_add',
                validate: validTemplateUpload
            });
        }

        //make sure we have at least 1 host
        if($scope.hosts.all && $scope.hosts.all.length === 0){
            $scope.newHost = {};
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
        for (var i=0; i < $scope.templates.data.length; i++) {
            var template = $scope.templates.data[i];
            if ($scope.install.selected[template.Id]) {
                templates[templates.length] = template;
            }
        }
        return templates;
    };

    $scope.getTemplateRequiredResources = function(template){
        var ret = {CPUCommitment:0, RAMCommitment:0};

        // if Services, iterate and sum up their commitment values
        if(template.Services){
            // recursively calculate cpu and ram commitments
            (function calcCommitment(services){
                services.forEach(function(service){
                    // CPUCommitment should be equal to max number of
                    // cores needed by any service
                    ret.CPUCommitment = Math.max(ret.CPUCommitment, service.CPUCommitment);
                    // RAMCommitment should be a sum of all ram needed
                    // by all services
                    ret.RAMCommitment += service.RAMCommitment;

                    // recurse!
                    if(service.Services) calcCommitment(service.Services);
                });
            })(template.Services);
        }

        return ret;
    };

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

        closeModal = function(){
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
                    resourcesService.get_deployed_templates(deploymentDefinition, function(data){
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

                        getStatus();
                    });
                }
            };

            $("#deployStatus").show();
            getStatus();
        }

        nextClicked = false;
    };

    resourcesService.get_app_templates(false, function(templatesMap) {
        var templates = [];
        for (var key in templatesMap) {
            var template = templatesMap[key];
            template.Id = key;
            templates.push(template);
        }
        $scope.templates.data = templates;
        refreshHosts($scope, resourcesService, true, resetStepPage);
    });

    refreshPools($scope, resourcesService, true);
}

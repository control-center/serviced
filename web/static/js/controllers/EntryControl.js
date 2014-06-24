function EntryControl($scope, authService, resourcesService) {
    authService.checkLogin($scope);
    $scope.brand_label = "brand_zcp";
    $scope.breadcrumbs = [{ label: 'breadcrumb_home', itemClass: 'active' }];
    $scope.page_content = "entry_content";
    $scope.showIfEmpty = function(){
        resourcesService.get_services(false, function(topServices, mappedServices){
            if( topServices.length <= 0 ){
                $('#addApp').modal('show');
            }
        });
    }
    resourcesService.get_version(function(data){
        $scope['version'] = data.Detail;
    });
}
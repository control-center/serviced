/* globals controlplane: true */
(function(){

    controlplane.factory("resourcesFactory", ["$http", "$location", "$notification", "DSCacheFactory", "$q", "$interval", "miscUtils",
    function($http, $location, $notification, DSCacheFactory, $q, $interval, utils) {
        // add function to $http service to allow for noCacheGet requests
        $http.noCacheGet = function(location){
          return $http({url: location, method: "GET", params: {'time': new Date().getTime()}});
        };

        var pollingFunctions = {};

        var redirectIfUnauthorized = function(status){
          if (status === 401) {
              utils.unauthorized($location);
          }
        };


        var methodConfigs = {
            assignIP: {
                method: "PUT",
                url: (id, ip) => `/services/${id}/ip/${ip}`,
            },
            getPools: {
                method: "GET",
                url: "/pools"
            },
            getPoolIPs: {
                method: "GET", 
                url: id => `/pools/${id}/ips`
            },
            addVHost: {
                method: "PUT",
                url: (serviceID, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                },
                payload: (serviceID, endpointName, vhostName) => {
                    return JSON.stringify({
                        'ServiceID': serviceID,
                        'Application': endpointName,
                        'VirtualHostName': vhostName
                    });
                }
            },
            deleteVHost: {
                method: "DELETE",
                url: (serviceID, endpointName, vhostName) => {
                    return `/services/${serviceID}/endpoint/${endpointName}/vhosts/${vhostName}`;
                }
            },
            getRunningServices: {
                method: "GET",
                url: "/running"
            },
            addPool: {
                method: "POST",
                url: "/pools/add",
                payload: pool => pool
            },
            removePool: {
                method: "DELETE",
                url: id => `/pools/${id}`
            },
            addPoolVirtualIP: {
                method: "PUT",
                url: poolID => `/pools/${poolID}/virtualip`,
                payload: (poolID, ip, netmask, bindInterface) => {
                    return JSON.stringify({
                        PoolID: poolID,
                        IP: ip,
                        Netmask: netmask,
                        BindInterface: bindInterface
                    });
                }
            },
            removePoolVirtualIP: {
                method: "DELETE",
                url: (poolID, ip) => `/pools/${poolID}/virtualip/${ip}`,
            },
            killRunning: {
                method: "DELETE",
                url: (hostID, instanceID) => `/hosts/${hostID}/${instanceID}`
            },
            getHosts: {
                method: "GET",
                url: "/hosts"
            },
            addHost: {
                method: "POST",
                url: "/hosts/add",
                payload: host => host
            },
            removeHost: {
                method: "DELETE",
                url: id => `/hosts/${id}`
            },
            getRunningHosts: {
                method: "GET",
                url: "/hosts/running"
            },
            getServices: {
                method: "GET",
                url: since => `/services${ since ? "?since="+ since : ""}`,
            },
            getInstanceLogs: {
                method: "GET",
                url: (serviceID, instanceID) => `/services/${serviceID}/${instanceID}/logs`
            },
            dockerIsLoggedIn: {
                method: "GET",
                url: "/dockerIsLoggedIn"
            },
            getAppTemplates: {
                method: "GET",
                url: "/templates"
            },
            removeAppTemplate: {
                method: "DELETE",
                url: id => `/templates/${id}`
            },
            updateService: {
                method: "PUT",
                url: id => `/services/${id}`,
                payload: (id, service) => service
            },
            deployAppTemplate: {
                method: "POST",
                url: "/templates/deploy",
                payload: template => template
            },
            removeService: {
                method: "DELETE",
                url: id => `/services/${id}`
            },
            startService: {
                method: "PUT",
                url: (id, skip) => `/services/${id}/startService${ skip ? "?auto=false" : "" }`
            },
            stopService: {
                method: "PUT",
                url: (id, skip) => `/services/${id}/stopService${ skip ? "?auto=false" : "" }`
            },
            restartService: {
                method: "PUT",
                url: (id, skip) => `/services/${id}/restartService${ skip ? "?auto=false" : "" }`
            },
            getVersion: {
                method: "GET",
                url: "/version"
            },
            getServiceHealth: {
                method: "GET",
                url: "/servicehealth"
            },
            getDeployStatus: {
                method: "POST",
                url: "/templates/deploy/status",
                payload: def => def
            },
            getDeployingTemplates: {
                method: "GET",
                url: "/templates/deploy/active"
            },
            createBackup: {
                method: "GET",
                url: "/backup/create"
            },
            restoreBackup: {
                method: "GET",
                url: filename => `/backup/restore?filename=${filename}`
            },
            getBackupFiles: {
                method: "GET",
                url: "/backup/list"
            },
            getBackupStatus: {
                method: "GET",
                url: "/backup/status"
            },
            getRestoreStatus: {
                method: "GET",
                url: "/backup/restore/status"
            },
            getHostAlias: {
                method: "GET",
                url: "/hosts/defaultHostAlias"
            }
        };

        function generateMethod(config){
            // method should be all lowercase
            config.method = config.method.toLowerCase();

            // if url is a string, turn it into a getter fn
            if(typeof config.url === "string"){
                let url = config.url;
                config.url = () => url;
            }

            return function(/* args */){ 
                var url = config.url.apply(null, arguments),
                    payload;

                if(config.payload){
                    payload = config.payload.apply(null, arguments);
                }

                return $http[config.method](url, payload)
                    .error(function(data, status) {
                        redirectIfUnauthorized(status);
                    });
            };
        }

        // generate methods from methodConfigs
        var methods = {};
        for(var name in methodConfigs){
            methods[name] = generateMethod(methodConfigs[name]);
        }

        return {
            assign_ip: methods.assignIP,
            get_pools: methods.getPools,
            // TODO - remove this and calculate values from servicesFactory
            get_pool_ips: methods.getPoolIPs,
            add_vhost: methods.addVHost,
            delete_vhost: methods.deleteVHost,
            get_running_services: methods.getRunningServices,
            add_pool: methods.addPool,
            remove_pool: methods.removePool,
            add_pool_virtual_ip: methods.addPoolVirtualIP,
            remove_pool_virtual_ip: methods.removePoolVirtualIP,
            kill_running: methods.killRunning,
            get_hosts: methods.getHosts,
            add_host: methods.addHost,
            remove_host: methods.removeHost,
            get_running_hosts: methods.getRunningHosts,
            get_services: methods.getServices,
            get_service_state_logs: methods.getInstanceLogs,
            docker_is_logged_in: methods.dockerIsLoggedIn,
            get_app_templates: methods.getAppTemplates,

            // TODO - templatize this guy?
            add_app_template: function(fileData){
              return $http({
                  url: "/templates/add",
                  method: "POST",
                  data: fileData,
                  // content-type undefined forces the browser to fill in the
                  // boundary parameter of the request
                  headers: {"Content-Type": undefined},
                  // identity returns the value it receives, which prevents
                  // transform from modifying the request in any way
                  transformRequest: angular.identity,
              }).error(function(data, status){
                  redirectIfUnauthorized(status);
              });
            },

            delete_app_template: methods.removeAppTemplate,
            update_service: methods.updateService,
            deploy_app_template: methods.deployAppTemplate,
            remove_service: methods.removeService,
            start_service: methods.startService,
            stop_service: methods.stopService,
            restart_service: methods.restartService,
            get_version: methods.getVersion,
            get_service_health: methods.getServiceHealth,
            get_deployed_templates: methods.getDeployStatus,
            get_active_templates: methods.getDeployingTemplates,
            create_backup: methods.createBackup,
            restore_backup: methods.restoreBackup,
            get_backup_files: methods.getBackupFiles,
            get_backup_status: methods.getBackupStatus,
            get_restore_status: methods.getRestoreStatus,
            get_host_alias: methods.getHostAlias,

            registerPoll: function(label, callback, interval){
              if(pollingFunctions[label] !== undefined){
                  clearInterval(pollingFunctions[label]);
              }

              pollingFunctions[label] = $interval(function(){
                  callback();
              }, interval);
            },

            unregisterAllPolls: function(){
              for(var key in pollingFunctions){
                  $interval.cancel(pollingFunctions[key]);
              }

              pollingFunctions = {};
            },

            // redirect to specific service details
            routeToService: function(id) {
                $location.path('/services/' + id);
            },

            // redirect to specific pool
            routeToPool: function(id) {
                $location.path('/pools/' + id);
            },

            // redirect to specific host
            routeToHost: function(id) {
                $location.path('/hosts/' + id);
            }
        };
    }]);
})();

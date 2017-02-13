/* jshint multistr: true */
(function () {
    'use strict';

    // share angular services outside of angular context

    console.log(" SERVICEACTIONS JS ");

    controlplane.factory('svcActions', SvcActionsFactory);

    class svcActions {

    }


    SvcActionsFactory.$inject = ['$serviceHealth'];
    function SvcActionsFactory(_serviceHealth) {

        // api access via angular context
        this.serviceHealth = _serviceHealth;

        return svcActions;

    }

})();
// An Angular factory that returns a mock implementation of the zenNotfy's NotificationFactory.
//
// To use this mock, call 'beforeEach(module(zenNotifyMock))' to inject this factory into
// Angular will then inject an instance of the spy created by this factory.
var zenNotifyMock = function($provide) {
    $provide.factory('$notification', function() {
        var mockNotification = jasmine.createSpyObj('Notification', [
            'success',
            'warning',
            'info',
            'error',
            'onClose',
            'hide',
            'updateStatus',
            'updateTitle',
            'show'
        ]);

        mockNotification.success.and.returnValue(mockNotification)
        mockNotification.warning.and.returnValue(mockNotification)
        mockNotification.info.and.returnValue(mockNotification)
        mockNotification.error.and.returnValue(mockNotification)
        mockNotification.updateStatus.and.returnValue(mockNotification)
        mockNotification.updateTitle.and.returnValue(mockNotification)
        mockNotification.show.and.returnValue(mockNotification)

        var mockNotificationFactory = jasmine.createSpyObj('NotificationFactory', [
            'create',
            'markRead',
            'store',
            'update',
            'getMessages',
            'clearAll'
        ]);
        mockNotificationFactory.create.and.returnValue(mockNotification);
        return mockNotificationFactory
    });
};

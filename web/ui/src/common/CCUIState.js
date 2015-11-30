/* CCUIState.js
 * preserve state through ui navigations
 */
(function(){
    "use strict";

    // TODO - persist to local storage
    class CCUIState {
        constructor(){
            // -> user name
            //    -> store name
            //       -> stored object
            this.store = {};
        }

        get(userName, storeName){
            var userStore = this.getUserStore(userName);

            // if the store doesnt exist for this user,
            // create it
            if(!userStore[storeName]){
                // TODO - formalize creation of this object
                userStore[storeName] = {};
            }

            return userStore[storeName];
        }

        // creates and returns a user store for the specified
        // user, or returns existing user store for the user
        getUserStore(name){
            var users = Object.keys(this.store);

            // if this user doesn't have a store,
            // create one
            if(users.indexOf(name) === -1){
                this.store[name] = {};
            }

            return this.store[name];
        }
    }

    angular.module("CCUIState", []).service("CCUIState", [CCUIState]);
})();

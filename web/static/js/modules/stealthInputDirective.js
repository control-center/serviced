/* stealthInput
 * a super fancy and adorable minimal input field
 */

(function() {
    'use strict';

    angular.module('stealthInput', []).
    directive("stealthInput", function(){
        return {
            restrict: "A",
            scope: {
                value: "=",
                onChange: "&",
                disabled: "&ngDisabled",
            },
            link: function(scope, inputElement, attr){
                var $input = inputElement,
                    $wrapper,
                    endEditTimeout;

                $wrapper = $("\
                    <div class='stealthInput'>\
                        <div class='dirty'></div>\
                        <div class='edit'></div>\
                        <div class='working'></div>\
                        <div class='controls'>\
                            <span class='cancel'></span>\
                            <span class='save'></span>\
                        </div>\
                    </div>\
                ");

                inputElement.before($wrapper);
                $wrapper.append($input);

                // if this field is disabled, dont do a thing
                scope.$watch(scope.disabled, function(value){
                    if(value){
                        console.log("disabled value is", value);
                        $wrapper.addClass("disabled");
                        $input.off("focus");
                    }
                });

                // if the model changes from the outside,
                // update the value in the input
                scope.$watch("value", function(value){
                    if(value !== undefined){
                        updateInputValue(value);
                    }
                });

                // add disable class to wrapper if input
                // is disabled
                scope.$watch("disabled", function(value){
                    if(value()){
                        $wrapper.addClass("disabled");
                    } else {
                        $wrapper.removeClass("disabled");
                    }
                });

                // wire up save and cancel controls
                $wrapper.find(".save").on("mousedown", function(){
                    clearTimeout(endEditTimeout);
                    commitChanges();
                    markIfDirty();
                });
                $wrapper.find(".cancel").on("mousedown", function(){
                    clearTimeout(endEditTimeout);
                    $input.val(scope.value);
                    markIfDirty();
                    endEdit();
                });

                // toggle edit mode
                $input.on("focus", function(){
                    startEdit();
                });
                $input.on("blur", function(){
                    // if we endEdit immediately on blur
                    // we might miss a click on save or cancel
                    // so wait a sec and then end edit
                    endEditTimeout = setTimeout(function(){
                        $wrapper.removeClass("isEditing");
                    }, 40);
                });
                $input.on("keyup", function(e){
                    var isEnter = e.keyCode === 13,
                        isEscape = e.keyCode === 27;

                    // if enter was pressed, save changes
                    if(isEnter){
                        commitChanges();

                    // if escape was pressed, revert changes
                    } else if(isEscape){
                        endEdit();
                        $input.val(scope.value);
                    }

                    // check dirty
                    markIfDirty();
                });
                $input.on("change", function(){
                    markIfDirty();
                });

                var isDirty = function(){
                    return scope.value != $input.val();
                };
                var markIfDirty = function(){
                    var dirty = isDirty();
                    
                    // make sure save/cancel controls are visible
                    // if this field is dirty
                    if(dirty && !$wrapper.hasClass("isDirty")){
                        $wrapper.addClass("isDirty");
                    } else if(!dirty){
                        $wrapper.removeClass("isDirty");
                    }

                    return dirty;
                };

                var startEdit = function(){
                    $wrapper.addClass("isEditing");
                };
                var endEdit = function(){
                    $wrapper.removeClass("isEditing");
                    $input.trigger("blur");
                };

                var startSpinning = function(){
                    $wrapper.addClass("isSpinning");
                };
                var endSpinning = function(){
                    $wrapper.removeClass("isSpinning");
                };

                var updateInputValue = function(val){
                    $input.val(val);
                    markIfDirty();
                };

                var commitChanges = function(){
                    // if no changes have been made, we done hurr.
                    if(!isDirty()) return;

                    var oldVal = scope.value,
                        val = $input.val(),
                        promise;

                    // coerce val to number if input type
                    // is number
                    if($input.attr("type") === "number"){
                        val = +val;
                    }
                   
                    scope.value = val;

                    // force watchers to be updated with the new value
                    scope.$apply();

                    endEdit();

                    if(scope.onChange()){
                        promise = scope.onChange()();

                        // if scope.onChange returns a promise, we can use
                        // it to add a helpful loading spinner
                        if(promise && typeof promise.then === "function"){
                            startSpinning();

                            promise.error(function(err){
                                scope.value = oldVal;
                                // need to wait a tick to do this so angular
                                // can get all the updaty stuff out of its system
                                setTimeout(function(){
                                    // put the dirty val back in the input
                                    $input.val(val);
                                    // dirty check
                                    markIfDirty();
                                }, 0);
                            })
                            .finally(function(){
                                endSpinning();
                            });
                        }
                    }
                };
            }
        };
    });

})();

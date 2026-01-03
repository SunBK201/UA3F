/**
 * UA3F Rule Manager
 */

(function (global) {
    'use strict';

    /**
     * RuleManager - Generic rule management component
     * @param {Object} config - Configuration object
     */
    function RuleManager(config) {
        this.config = Object.assign({
            // Required
            containerId: '',
            tableId: '',
            tbodyId: '',
            saveUrl: '',

            // Rule configuration
            ruleKey: 'rules',
            initialRules: [],

            // Labels (i18n)
            labels: {},

            // Rule types configuration
            ruleTypes: [],

            // Action types configuration
            actionTypes: [],
            finalActionTypes: [],

            // Direction types
            directionTypes: [
                { value: 'REQUEST', label: 'HTTP Request' },
                { value: 'RESPONSE', label: 'HTTP Response' }
            ],

            // Table columns configuration
            columns: [],

            // Dialog fields configuration
            dialogFields: [],

            // Callbacks
            onBeforeSave: null,
            onAfterSave: null,
            onValidate: null,

            // Features
            hasFinalRule: true,
            allowMove: true,
            allowDelete: true,
            allowToggle: true
        }, config);

        this.rules = this.config.initialRules.slice();
        this.currentModal = null;

        this.init();
    }

    RuleManager.prototype = {
        /**
         * Initialize the rule manager
         */
        init: function () {
            if (this.config.hasFinalRule) {
                this.ensureFinalRule();
            }
            this.renderTable();
            this.initDragAndDrop();
        },

        /**
         * Initialize drag and drop functionality
         */
        initDragAndDrop: function () {
            var self = this;
            this.dragState = {
                dragging: false,
                draggedIndex: -1,
                draggedRow: null,
                placeholder: null
            };
        },

        /**
         * Handle drag start
         */
        handleDragStart: function (e, index) {
            var self = this;
            if (this.isFinalRule(index)) {
                e.preventDefault();
                return;
            }

            this.dragState.dragging = true;
            this.dragState.draggedIndex = index;
            this.dragState.draggedRow = e.target.closest('tr');

            if (this.dragState.draggedRow) {
                this.dragState.draggedRow.classList.add('dragging');
            }

            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/plain', index.toString());
        },

        /**
         * Handle drag over
         */
        handleDragOver: function (e, index) {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';

            var targetRow = e.target.closest('tr');
            if (!targetRow) return;

            // Don't allow dropping on FINAL rule or after it
            if (this.isFinalRule(index)) {
                return;
            }

            var tbody = document.getElementById(this.config.tbodyId);
            var rows = tbody.querySelectorAll('tr.cbi-section-table-row');

            rows.forEach(function (row) {
                row.classList.remove('drag-over-top', 'drag-over-bottom');
            });

            var rect = targetRow.getBoundingClientRect();
            var midY = rect.top + rect.height / 2;

            if (e.clientY < midY) {
                targetRow.classList.add('drag-over-top');
            } else {
                targetRow.classList.add('drag-over-bottom');
            }
        },

        /**
         * Handle drag leave
         */
        handleDragLeave: function (e) {
            var targetRow = e.target.closest('tr');
            if (targetRow) {
                targetRow.classList.remove('drag-over-top', 'drag-over-bottom');
            }
        },

        /**
         * Handle drop
         */
        handleDrop: function (e, targetIndex) {
            e.preventDefault();
            var self = this;

            var tbody = document.getElementById(this.config.tbodyId);
            var rows = tbody.querySelectorAll('tr.cbi-section-table-row');
            rows.forEach(function (row) {
                row.classList.remove('drag-over-top', 'drag-over-bottom', 'dragging');
            });

            var sourceIndex = this.dragState.draggedIndex;
            if (sourceIndex === -1 || sourceIndex === targetIndex) {
                this.resetDragState();
                return;
            }

            // Don't allow dropping on or after FINAL rule
            if (this.isFinalRule(targetIndex)) {
                this.resetDragState();
                return;
            }

            // Calculate the actual target position based on drop position
            var targetRow = e.target.closest('tr');
            var rect = targetRow.getBoundingClientRect();
            var dropAfter = e.clientY > rect.top + rect.height / 2;

            // Move the rule
            var rule = this.rules.splice(sourceIndex, 1)[0];
            var newIndex = targetIndex;
            if (sourceIndex < targetIndex) {
                newIndex = dropAfter ? targetIndex : targetIndex - 1;
            } else {
                newIndex = dropAfter ? targetIndex + 1 : targetIndex;
            }

            // Ensure we don't place after FINAL rule
            var maxIndex = this.config.hasFinalRule ? this.rules.length - 1 : this.rules.length;
            if (newIndex > maxIndex) {
                newIndex = maxIndex;
            }

            this.rules.splice(newIndex, 0, rule);
            this.resetDragState();
            this.saveRules();
        },

        /**
         * Handle drag end
         */
        handleDragEnd: function (e) {
            var tbody = document.getElementById(this.config.tbodyId);
            if (tbody) {
                var rows = tbody.querySelectorAll('tr.cbi-section-table-row');
                rows.forEach(function (row) {
                    row.classList.remove('drag-over-top', 'drag-over-bottom', 'dragging');
                });
            }
            this.resetDragState();
        },

        /**
         * Reset drag state
         */
        resetDragState: function () {
            this.dragState.dragging = false;
            this.dragState.draggedIndex = -1;
            this.dragState.draggedRow = null;
        },

        /**
         * Ensure FINAL rule exists and is at the bottom
         */
        ensureFinalRule: function () {
            var self = this;
            var hasFinalRule = this.rules.some(function (rule) {
                return rule.type === 'FINAL';
            });

            if (!hasFinalRule) {
                this.rules.push({
                    type: 'FINAL',
                    match_value: '',
                    action: 'DIRECT',
                    rewrite_value: '',
                    description: this.config.labels.defaultFinalDescription || 'Default fallback rule',
                    enabled: true
                });
            } else {
                var finalRuleIndex = -1;
                for (var i = 0; i < this.rules.length; i++) {
                    if (this.rules[i].type === 'FINAL') {
                        finalRuleIndex = i;
                        break;
                    }
                }
                if (finalRuleIndex !== -1 && finalRuleIndex !== this.rules.length - 1) {
                    var finalRule = this.rules.splice(finalRuleIndex, 1)[0];
                    this.rules.push(finalRule);
                }
            }
        },

        /**
         * Check if rule at index is FINAL
         */
        isFinalRule: function (index) {
            return this.rules[index] && this.rules[index].type === 'FINAL';
        },

        /**
         * Render the rules table
         */
        renderTable: function () {
            var tbody = document.getElementById(this.config.tbodyId);
            if (!tbody) return;

            tbody.innerHTML = '';

            if (this.rules.length === 0) {
                var tr = document.createElement('tr');
                tr.className = 'tr';
                var td = document.createElement('td');
                td.className = 'td empty-message';
                td.colSpan = this.config.columns.length;
                td.textContent = this.config.labels.emptyMessage || 'No rules configured';
                tr.appendChild(td);
                tbody.appendChild(tr);
                return;
            }

            for (var i = 0; i < this.rules.length; i++) {
                tbody.appendChild(this.createRuleRow(this.rules[i], i));
            }
        },

        /**
         * Create a table row for a rule
         */
        createRuleRow: function (rule, index) {
            var self = this;
            var tr = document.createElement('tr');
            tr.className = 'tr cbi-section-table-row';
            var isFinal = this.isFinalRule(index);

            // Add drag attributes
            if (this.config.allowMove && !isFinal) {
                tr.draggable = true;
                tr.ondragstart = function (e) { self.handleDragStart(e, index); };
                tr.ondragend = function (e) { self.handleDragEnd(e); };
            }
            tr.ondragover = function (e) { self.handleDragOver(e, index); };
            tr.ondragleave = function (e) { self.handleDragLeave(e); };
            tr.ondrop = function (e) { self.handleDrop(e, index); };

            // Add drag handle as first column if allowMove is enabled
            if (this.config.allowMove) {
                var dragTd = document.createElement('td');
                dragTd.className = 'td drag-handle-cell';
                dragTd.style.width = '30px';
                dragTd.style.textAlign = 'center';

                if (!isFinal) {
                    var dragHandle = document.createElement('span');
                    dragHandle.className = 'drag-handle';
                    dragHandle.title = 'Drag to reorder';
                    // Use inline SVG for better compatibility
                    dragHandle.innerHTML = '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><rect x="2" y="3" width="12" height="2" rx="1"/><rect x="2" y="7" width="12" height="2" rx="1"/><rect x="2" y="11" width="12" height="2" rx="1"/></svg>';
                    dragTd.appendChild(dragHandle);
                }
                tr.appendChild(dragTd);
            }

            this.config.columns.forEach(function (column) {
                var td = document.createElement('td');
                td.className = 'td';

                if (column.style) {
                    Object.keys(column.style).forEach(function (key) {
                        td.style[key] = column.style[key];
                    });
                }

                switch (column.type) {
                    case 'checkbox':
                        td.style.textAlign = 'center';
                        var checkbox = document.createElement('input');
                        checkbox.type = 'checkbox';
                        checkbox.className = 'cbi-input-checkbox';
                        checkbox.checked = rule.enabled;
                        if (isFinal) {
                            checkbox.disabled = true;
                            checkbox.checked = true;
                        } else if (self.config.allowToggle) {
                            checkbox.onchange = function () {
                                self.toggleRuleEnabled(index, this.checked);
                            };
                        }
                        td.appendChild(checkbox);
                        break;

                    case 'index':
                        td.textContent = index + 1;
                        break;

                    case 'label':
                        var labelFn = column.labelMap || function (v) { return v; };
                        td.textContent = labelFn.call(self, rule[column.field]);
                        break;

                    case 'value':
                        td.style.maxWidth = column.maxWidth || '150px';
                        td.style.overflow = 'hidden';
                        td.style.textOverflow = 'ellipsis';
                        td.style.whiteSpace = 'nowrap';
                        var span = document.createElement('span');
                        var value = isFinal && column.hideForFinal ? '-' : (rule[column.field] || '');
                        span.textContent = value;
                        span.title = isFinal && column.hideForFinal ? '' : (rule[column.field] || '');
                        td.appendChild(span);
                        break;

                    case 'actions':
                        td.className = 'td rule-actions';
                        self.createActionButtons(td, index, isFinal);
                        break;
                }

                tr.appendChild(td);
            });

            return tr;
        },

        /**
         * Create action buttons for a rule row
         */
        createActionButtons: function (td, index, isFinal) {
            var self = this;

            // Edit button
            var editBtn = document.createElement('button');
            editBtn.type = 'button';
            editBtn.className = 'cbi-button cbi-button-edit';
            editBtn.textContent = this.config.labels.edit || 'Edit';
            editBtn.onclick = function () { self.editRule(index); };
            td.appendChild(editBtn);

            if (!isFinal) {
                if (this.config.allowDelete) {
                    td.appendChild(document.createTextNode(' '));

                    var delBtn = document.createElement('button');
                    delBtn.type = 'button';
                    delBtn.className = 'cbi-button cbi-button-remove';
                    delBtn.textContent = this.config.labels.delete || 'Delete';
                    delBtn.onclick = function () { self.deleteRule(index); };
                    td.appendChild(delBtn);
                }
            }
        },

        /**
         * Get label for rule type
         */
        getRuleTypeLabel: function (type) {
            for (var i = 0; i < this.config.ruleTypes.length; i++) {
                if (this.config.ruleTypes[i].value === type) {
                    return this.config.ruleTypes[i].label;
                }
            }
            if (type === 'FINAL') {
                return this.config.labels.final || 'FINAL';
            }
            return type;
        },

        /**
         * Get label for action
         */
        getActionLabel: function (action) {
            var allActions = this.config.actionTypes.concat(this.config.finalActionTypes || []);
            for (var i = 0; i < allActions.length; i++) {
                if (allActions[i].value === action) {
                    return allActions[i].label;
                }
            }
            return action;
        },

        /**
         * Open add rule dialog
         */
        openAddDialog: function () {
            this.showRuleDialog(null, -1);
        },

        /**
         * Edit existing rule
         */
        editRule: function (index) {
            this.showRuleDialog(this.rules[index], index);
        },

        /**
         * Show rule dialog (add/edit)
         */
        showRuleDialog: function (rule, index) {
            var self = this;
            var isEdit = rule !== null;
            var isFinal = isEdit && rule.type === 'FINAL';

            // Create modal structure
            var modal = document.createElement('div');
            modal.className = 'cbi-modal';

            var dialog = document.createElement('div');
            dialog.className = 'cbi-modal-dialog';

            // Header
            var header = document.createElement('div');
            header.className = 'cbi-modal-dialog-header';
            var h3 = document.createElement('h3');
            h3.textContent = isFinal
                ? (this.config.labels.editFinalRule || 'Edit FINAL Rule')
                : (isEdit
                    ? (this.config.labels.editRule || 'Edit Rule')
                    : (this.config.labels.addRule || 'Add Rule'));
            header.appendChild(h3);
            dialog.appendChild(header);

            // Body
            var body = document.createElement('div');
            body.className = 'cbi-modal-dialog-body';

            var section = document.createElement('div');
            section.className = 'cbi-section';

            // Render dialog fields
            this.config.dialogFields.forEach(function (field) {
                // Check if field should be shown
                if (field.hideForFinal && isFinal) return;
                if (field.showOnlyForFinal && !isFinal) return;

                var fieldElement = self.createDialogField(field, rule, isFinal);
                if (fieldElement) {
                    section.appendChild(fieldElement);
                }
            });

            body.appendChild(section);

            // Buttons
            var btnDiv = document.createElement('div');
            btnDiv.className = 'right';

            var cancelBtn = document.createElement('button');
            cancelBtn.type = 'button';
            cancelBtn.className = 'cbi-button cbi-button-neutral';
            cancelBtn.textContent = this.config.labels.cancel || 'Cancel';
            cancelBtn.onclick = function () { self.closeDialog(); };
            btnDiv.appendChild(cancelBtn);

            btnDiv.appendChild(document.createTextNode(' '));

            var saveBtn = document.createElement('button');
            saveBtn.type = 'button';
            saveBtn.className = 'cbi-button cbi-button-positive';
            saveBtn.textContent = this.config.labels.save || 'Save';
            saveBtn.onclick = function () { self.saveFromDialog(index); };
            btnDiv.appendChild(saveBtn);

            body.appendChild(btnDiv);
            dialog.appendChild(body);
            modal.appendChild(dialog);

            modal.onclick = function (e) {
                if (e.target === modal) self.closeDialog();
            };

            document.body.appendChild(modal);
            this.currentModal = modal;

            // Initialize field visibility
            this.updateFieldVisibility();
        },

        /**
         * Create a dialog field element
         */
        createDialogField: function (fieldConfig, rule, isFinal) {
            var self = this;
            var container = document.createElement('div');
            container.className = 'cbi-value';
            if (fieldConfig.id) {
                container.id = fieldConfig.id + '_container';
            }

            // Label
            var label = document.createElement('label');
            label.className = 'cbi-value-title';
            label.textContent = fieldConfig.label + (fieldConfig.optional ? ' (' + (this.config.labels.optional || 'Optional') + ')' : '');
            container.appendChild(label);

            // Field
            var fieldDiv = document.createElement('div');
            fieldDiv.className = 'cbi-value-field';

            var element;
            var currentValue = rule ? rule[fieldConfig.field] : fieldConfig.defaultValue;

            switch (fieldConfig.type) {
                case 'select':
                    element = document.createElement('select');
                    element.className = 'cbi-input-select';

                    var options = fieldConfig.optionsKey
                        ? this.config[fieldConfig.optionsKey]
                        : fieldConfig.options;

                    // For FINAL rule actions, use finalActionTypes if available
                    if (fieldConfig.field === 'action' && isFinal && this.config.finalActionTypes) {
                        options = this.config.finalActionTypes;
                    }

                    options.forEach(function (opt) {
                        var option = document.createElement('option');
                        option.value = opt.value;
                        option.textContent = opt.label;
                        if (currentValue === opt.value) option.selected = true;
                        element.appendChild(option);
                    });

                    if (fieldConfig.onChange) {
                        element.onchange = function () {
                            fieldConfig.onChange.call(self, this.value);
                        };
                    }
                    break;

                case 'text':
                    element = document.createElement('input');
                    element.type = 'text';
                    element.className = 'cbi-input-text';
                    element.placeholder = fieldConfig.placeholder || '';
                    element.value = currentValue || fieldConfig.defaultValue || '';
                    break;

                case 'checkbox':
                    fieldDiv.style.display = 'flex';
                    fieldDiv.style.alignItems = 'center';
                    element = document.createElement('input');
                    element.type = 'checkbox';
                    element.className = 'cbi-input-checkbox';
                    element.checked = currentValue || false;
                    break;
            }

            if (element) {
                element.id = 'modal_' + fieldConfig.field;
                fieldDiv.appendChild(element);
            }

            container.appendChild(fieldDiv);
            return container;
        },

        /**
         * Update field visibility based on current selections
         */
        updateFieldVisibility: function () {
            var self = this;

            this.config.dialogFields.forEach(function (field) {
                if (field.visibilityRules) {
                    var container = document.getElementById(field.id + '_container');
                    if (!container) return;

                    var shouldShow = true;
                    field.visibilityRules.forEach(function (rule) {
                        var targetEl = document.getElementById('modal_' + rule.field);
                        if (targetEl) {
                            var value = targetEl.value;
                            if (rule.showWhen && rule.showWhen.indexOf(value) === -1) {
                                shouldShow = false;
                            }
                            if (rule.hideWhen && rule.hideWhen.indexOf(value) !== -1) {
                                shouldShow = false;
                            }
                        }
                    });

                    container.style.display = shouldShow ? '' : 'none';
                }
            });
        },

        /**
         * Close the dialog
         */
        closeDialog: function () {
            if (this.currentModal) {
                document.body.removeChild(this.currentModal);
                this.currentModal = null;
            }
        },

        /**
         * Save rule from dialog
         */
        saveFromDialog: function (index) {
            var self = this;
            var isFinal = index >= 0 && this.rules[index].type === 'FINAL';

            var newRule = {};

            // Collect values from dialog fields
            this.config.dialogFields.forEach(function (field) {
                var element = document.getElementById('modal_' + field.field);
                if (!element) return;

                var container = document.getElementById(field.id + '_container');
                if (container && container.style.display === 'none') return;

                var value;
                if (field.type === 'checkbox') {
                    value = element.checked;
                } else {
                    value = element.value;
                }

                newRule[field.field] = value;
            });

            // Set type for FINAL rules
            if (isFinal) {
                newRule.type = 'FINAL';
            }

            // Preserve enabled state
            newRule.enabled = index >= 0 ? this.rules[index].enabled : true;

            // Custom validation
            if (this.config.onValidate) {
                var error = this.config.onValidate.call(this, newRule, isFinal);
                if (error) {
                    alert(error);
                    return;
                }
            }

            // Before save callback
            if (this.config.onBeforeSave) {
                newRule = this.config.onBeforeSave.call(this, newRule, isFinal) || newRule;
            }

            // Update rules array
            if (index >= 0) {
                this.rules[index] = newRule;
            } else {
                // Insert at the beginning of the list
                this.rules.unshift(newRule);
            }

            this.closeDialog();
            this.saveRules();
        },

        /**
         * Delete a rule
         */
        deleteRule: function (index) {
            if (this.isFinalRule(index)) {
                alert(this.config.labels.cannotDeleteFinal || 'FINAL rule cannot be deleted');
                return;
            }

            if (!confirm(this.config.labels.confirmDelete || 'Are you sure you want to delete this rule?')) {
                return;
            }

            this.rules.splice(index, 1);
            this.saveRules();
        },

        /**
         * Move rule up
         */
        moveRuleUp: function (index) {
            if (index > 0 && !this.isFinalRule(index)) {
                var temp = this.rules[index];
                this.rules[index] = this.rules[index - 1];
                this.rules[index - 1] = temp;
                this.saveRules();
            }
        },

        /**
         * Move rule down
         */
        moveRuleDown: function (index) {
            if (index < this.rules.length - 2 && !this.isFinalRule(index)) {
                var temp = this.rules[index];
                this.rules[index] = this.rules[index + 1];
                this.rules[index + 1] = temp;
                this.saveRules();
            }
        },

        /**
         * Toggle rule enabled state
         */
        toggleRuleEnabled: function (index, enabled) {
            this.rules[index].enabled = enabled;
            this.saveRules(function (success) {
                if (!success) {
                    // Revert on failure
                    this.rules[index].enabled = !enabled;
                    this.renderTable();
                }
            }.bind(this));
        },

        /**
         * Save rules to server
         */
        saveRules: function (callback) {
            var self = this;
            var xhr = new XMLHttpRequest();
            xhr.open('POST', this.config.saveUrl);
            xhr.setRequestHeader('Content-Type', 'application/json');
            xhr.onload = function () {
                var success = xhr.status === 200;
                if (success) {
                    self.renderTable();
                    if (self.config.onAfterSave) {
                        self.config.onAfterSave.call(self, true);
                    }
                } else {
                    alert(self.config.labels.saveFailed || 'Failed to save rules');
                    if (self.config.onAfterSave) {
                        self.config.onAfterSave.call(self, false);
                    }
                }
                if (callback) callback(success);
            };

            var data = {};
            data[this.config.ruleKey] = this.rules;
            xhr.send(JSON.stringify(data));
        }
    };

    // Export to global scope
    global.RuleManager = RuleManager;

})(window);

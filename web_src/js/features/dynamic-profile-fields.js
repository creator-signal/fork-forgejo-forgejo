// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

function initDynamicProfileFields() {
  const containers = document.querySelectorAll('.dynamic-fields-container');
  if (!containers.length) return;

  for (const container of containers) {
    const addButton = container.querySelector('.add-field-button');
    const template = container.querySelector('template');
    const fieldList = container.querySelector('.field-list');
    const fieldPrefix = container.dataset.prefix;

    const updateFieldNames = () => {
      const rows = fieldList.querySelectorAll('.field-group-row');
      rows.forEach((row, index) => {
        const nameInput = row.querySelector('input[data-name="name"]');
        const valueInput = row.querySelector('input[data-name="value"]');
        if (nameInput) nameInput.name = `${fieldPrefix}_name[]`;
        if (valueInput) valueInput.name = `${fieldPrefix}_value[]`;
      });
    };

    const addField = () => {
      const content = template.content.cloneNode(true);
      const newRow = content.querySelector('.field-group-row');
      fieldList.append(newRow);
      const removeButton = newRow.querySelector('.remove-field-button');
      removeButton.addEventListener('click', () => {
        newRow.remove();
        updateFieldNames();
      });
      updateFieldNames();
    };

    addButton.addEventListener('click', addField);

    // Add remove listeners to existing fields
    const existingRows = fieldList.querySelectorAll('.field-group-row');
    for (const row of existingRows) {
      const removeButton = row.querySelector('.remove-field-button');
      if (removeButton) {
        removeButton.addEventListener('click', () => {
          row.remove();
          updateFieldNames();
        });
      }
    }
  }
}

document.addEventListener('DOMContentLoaded', initDynamicProfileFields);

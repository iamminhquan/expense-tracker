// The categories page's mobile add-category sheet.
//
// There is one add-category form on the page, and it moves: on desktop it
// sits in the sidebar slot, and on mobile the sticky header's + button pulls
// that same element into the bottom sheet, then puts it back when the sheet
// closes. Moving the node rather than rendering the form twice is what keeps
// a validation error, a typed name, and the hx-post target in one place.
//
// Loaded by a <script src> at the end of the categories content block rather
// than from <head>, so an hx-boost navigation onto this page re-runs it and
// rebinds against the DOM it just swapped in. openAddCategorySheet is global
// because the + button lives in the shared mobile header (nav.html), which
// knows nothing about this page beyond the function's name.

function openAddCategorySheet() {
  var slot = document.getElementById('add-category-slot');
  var body = document.getElementById('add-category-sheet-body');
  if (slot.firstElementChild) body.appendChild(slot.firstElementChild);
  document.getElementById('add-category-sheet').showModal();
}

document.getElementById('add-category-sheet').addEventListener('close', function () {
  var slot = document.getElementById('add-category-slot');
  var body = document.getElementById('add-category-sheet-body');
  if (body.firstElementChild) slot.appendChild(body.firstElementChild);
});

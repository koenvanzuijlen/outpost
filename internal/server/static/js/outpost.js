// A select with the data-auto-submit attribute submits its form when the value changes.
document.addEventListener("change", function (event) {
  if (
    event.target instanceof HTMLSelectElement &&
    event.target.matches("[data-auto-submit]")
  ) {
    event.target.form?.requestSubmit();
  }
});

// A select with the data-auto-submit attribute submits its form when the value changes.
document.addEventListener("change", function (event) {
  if (
    event.target instanceof HTMLSelectElement &&
    event.target.matches("[data-auto-submit]")
  ) {
    event.target.form?.requestSubmit();
  }
});

// saveQueues holds the last save of each form with data-auto-save. A new save starts after the previous one ends, so
// the requests reach the server in order and the last request carries the latest form state.
const saveQueues = new WeakMap();

// saveForm posts all fields of form to its action and shows the result in the form's [data-save-status] element. The
// server answers a successful save with a redirect, which fetch does not follow. The function never rejects, so a
// failed save does not stop later saves in the queue. It does nothing if the form has no [data-save-status] element.
async function saveForm(form) {
  const statusField = form.querySelector("[data-save-status]");
  if (!statusField) {
    return;
  }
  statusField.textContent = "Saving…";
  try {
    const response = await fetch(form.action, {
      method: "POST",
      body: new URLSearchParams(new FormData(form)),
      redirect: "manual",
      keepalive: true,
    });
    if (response.type === "opaqueredirect") {
      statusField.textContent = "Saved.";
    } else {
      statusField.textContent = "Couldn't save. Reload the page and try again.";
    }
  } catch {
    statusField.textContent = "Couldn't save. Reload the page and try again.";
  }
}

// A form with the data-auto-save attribute saves when an input or textarea changes. A textarea changes when it loses
// focus after an edit.
document.addEventListener("change", function (event) {
  const form = event.target.form;
  if (
    form?.matches("[data-auto-save]") &&
    (event.target instanceof HTMLInputElement ||
      event.target instanceof HTMLTextAreaElement)
  ) {
    saveQueues.set(
      form,
      (saveQueues.get(form) ?? Promise.resolve()).then(() => saveForm(form)),
    );
  }
});

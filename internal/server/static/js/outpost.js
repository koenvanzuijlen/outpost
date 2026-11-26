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

// showSignedOut shows in statusField that the session has ended, with a link to the sign-in page. After sign-in, the
// sign-in page returns to the current page.
function showSignedOut(statusField) {
  const link = document.createElement("a");
  const query = new URLSearchParams({
    next: window.location.pathname + window.location.search,
  });
  link.href = "/sign-in?" + query;
  link.textContent = "Sign in";

  statusField.replaceChildren(
    "You are signed out. ",
    link,
    " to save your changes.",
  );
}

// saveForm posts all fields of form to its action and shows the result in the form's [data-save-status] element. The
// server answers a successful save with a redirect, which fetch does not follow, and a request without a valid
// session with a 401. The function never rejects, so a failed save does not stop later saves in the queue. It does
// nothing if the form has no [data-save-status] element.
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
    } else if (response.status === 401) {
      showSignedOut(statusField);
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

// updateSelectionCount shows the number of checked games of form in its [data-selection-count] element. It does nothing
// if the form has no such element.
function updateSelectionCount(form) {
  const countField = form.querySelector("[data-selection-count]");
  if (!countField) {
    return;
  }
  const count = form.querySelectorAll('input[name="igdb_id"]:checked').length;
  countField.textContent = `${count} selected`;
}

// A button with data-select="all" or "none" checks or unchecks every game of its form. Setting checked fires no change
// event, so the count is updated here.
document.addEventListener("click", function (event) {
  const button = event.target.closest("[data-select]");
  if (!button) {
    return;
  }
  const form = button.form;
  if (!form?.matches("[data-selection]")) {
    return;
  }
  for (const input of form.querySelectorAll('input[name="igdb_id"]')) {
    input.checked = button.dataset.select === "all";
  }
  updateSelectionCount(form);
});

// A change of a game checkbox in a form with data-selection updates the count of its form.
document.addEventListener("change", function (event) {
  if (event.target.matches('input[name="igdb_id"]')) {
    const form = event.target.form;
    if (form?.matches("[data-selection]")) {
      updateSelectionCount(form);
    }
  }
});

// Shows the selection tools of each form with data-selection, and its first count. The browser can restore the checks
// on a back navigation, so the first count is not always 0.
for (const form of document.querySelectorAll("form[data-selection]")) {
  form.querySelector("[data-selection-tools]")?.removeAttribute("hidden");
  updateSelectionCount(form);
}

// Fills the time zone suggestions of the settings page with the time zones that the browser knows.
for (const list of document.querySelectorAll("datalist[data-time-zones]")) {
  for (const tz of Intl.supportedValuesOf("timeZone")) {
    list.appendChild(new Option(tz));
  }
}

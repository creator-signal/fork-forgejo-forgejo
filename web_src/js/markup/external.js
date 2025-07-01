export function renderExternal() {
  const giteaExternalRender = document.querySelector('iframe.external-render');
  if (!giteaExternalRender) return;

  const eventListener = (event) => {
    if (event.source != giteaExternalRender.contentWindow) return;
    const height = Number(event.data && event.data.frameHeight);
    if (!height) return;
    giteaExternalRender.height = height;
    giteaExternalRender.style.overflow = "hidden";
    window.removeEventListener("message", eventListener);
  };
  window.addEventListener("message", eventListener);
}

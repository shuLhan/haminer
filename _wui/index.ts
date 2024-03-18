class Haminer {
  apiLogTail(id: string) {
    var comp = document.getElementById(id);

    const evtSource = new EventSource("/api/log/tail");

    evtSource.onmessage = (event) => {
      const elLog = document.createElement("div");

      console.log(`${event.data}`);

      elLog.textContent = event.data;

      comp.prepend(elLog);
    };
  }
}

let haminer = new Haminer();

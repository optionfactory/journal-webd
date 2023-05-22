class JournalViewer {
    constructor(conf) {
        const ahs = document.querySelector('#allowedhosts');
        conf.hosts.forEach(h => {
            const opt = document.createElement('option');
            opt.value=h
            ahs.append(opt);
        })
        const aus = document.querySelector('#allowedunits');
        conf.units.forEach(u => {
            const opt = document.createElement('option');
            opt.value=u
            aus.append(opt);
        })
        this.hosts = document.querySelector('#hosts');
        this.units = document.querySelector('#units');
        this.filter = document.querySelector('#filter');
        this.range = document.querySelector('#range');
        this.follow = document.querySelector('#follow');
        this.logstablebody = document.querySelector('#logs > tbody');
        this.startButton = document.querySelector('[data-ref=start]');
        this.stopButton = document.querySelector('[data-ref=stop]');

        const ee = new ftl.ExpressionEvaluator({
            json: {
                stringify: JSON.stringify
            } 
        });
        const tnee = new ftl.TextNodeExpressionEvaluator(ee);
        const ch = new ftl.TplCommandsHandler();

        this.rowTemplate = ftl.Template.fromNode(document.querySelector('#table-row-template').content, {
            evaluator: ee,
            textNodeEvaluator: tnee,
            commandsHandler: ch
        });
        this.queue = []
        this.renderBufferedData = ful.timing.throttle(100, this.renderBufferedData.bind(this));
    }
    started() {
        this.hosts.disabled = true;
        this.units.disabled = true
        this.follow.disabled = true;
        this.range.disabled = true;

        this.clearRows();

        this.startButton.classList.add('d-none');
        this.stopButton.classList.remove('d-none');
    }
    stopped() {
        this.hosts.disabled = false;
        this.units.disabled = false;
        this.follow.disabled = false;
        this.range.disabled = false;

        this.startButton.classList.remove('d-none');
        this.stopButton.classList.add('d-none');
    }
    clearRows() {
        this.logstablebody.innerHTML = '';
    }
    pushRowData(data){
        this.queue.push(data);
        this.renderBufferedData();
    }
    renderBufferedData() {
        const rows = this.queue.map(data => {
            const row = JSON.parse(data);
            return {
                hostname: row._HOSTNAME || "",
                pid: row._PID || "",
                unit: row._SYSTEMD_UNIT || "",
                timestamp: new Date(+row.__REALTIME_TIMESTAMP / 1000).toLocaleString(),
                message: row.MESSAGE || ""
            }
        });
        this.rowTemplate.appendTo(this.logstablebody, rows);
        this.queue = [];
    }
}


function each(selector, cb){
    document.querySelectorAll(selector).forEach(cb);
}

function on(selector, event, cb){
    document.querySelectorAll(selector).forEach(el => el.addEventListener(event, cb));
}

document.addEventListener("DOMContentLoaded", async () => {
    on('[data-column-toggle]', 'change', event => {
        document.body.classList[event.target.checked ? 'remove' : 'add']('hide-' + event.target.dataset.columnToggle);
    });

    on('[data-range-type]', 'click', (event) => {
        viewer.range.innerText = event.target.innerText;
        viewer.range.dataset.rangeType = event.target.dataset.rangeType;
        viewer.range.dataset.rangeOptions = event.target.dataset.rangeOptions;
        //TODO: if timespanType==timespan_period
    });


    const auth = new Authorization();
    await auth.setup();

    on('[data-ref=logout]', 'click', () => {
        auth.logout();
    });

    const resp = await auth.http.fetch('/api/conf', {});
    const conf = await resp.json();

    const viewer = new JournalViewer(conf);

    let socket = null;

    on('[data-ref=clear]', 'click', viewer.clearRows.bind(viewer));

    on('[data-ref=stop]', 'click', () => {
        if (socket) {
            socket.close(1000, 'Work complete');
        }
        viewer.stopped();
    });

    on('[data-ref=start]', 'click', () => {
        if (socket) {
            socket.close(1000, 'Work complete');
        }
        viewer.started();
        socket = new WebSocket(`${window.location.protocol == 'https:' ? 'wss:': 'ws:'}//${window.location.host}/ws/stream`);
        socket.addEventListener("open", async (event) => {
            await auth.authorizeWebSocket(socket);
            socket.send(JSON.stringify({
                hosts: viewer.hosts.value.split(",").map(v => v.trim()).filter(v => v.length !== 0).filter((v,i,a) => a.indexOf(v) === i),
                units: viewer.units.value.split(",").map(v => v.trim()).filter(v => v.length !== 0).filter((v,i,a) => a.indexOf(v) === i), 
                ["range_"+viewer.range.dataset.rangeType]: {...JSON.parse(viewer.range.dataset.rangeOptions), follow: viewer.follow.checked},
                filter: viewer.filter.value
            }));
        });
        socket.addEventListener("close", viewer.stopped.bind(viewer));
        socket.addEventListener("message", (event) => {
            viewer.pushRowData(event.data);
        });
    });

    document.querySelector('#loader').classList.add('d-none');
    document.querySelector('table').classList.remove('d-none');
});

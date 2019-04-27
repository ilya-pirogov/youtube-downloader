package pkg

import (
	"html/template"
	"net/http"
)

const indexHtml = `<html lang="en">
<head>
    <title>Youtube MP3 Downloader</title>
    <style>
        body {
           font-family: 'Georgia', serif;
			margin: 30px;
        }
        
        #root {
            display: flex;
            flex-wrap: wrap;
        }

        .panel {
            margin-bottom: 15px;
        }
    </style>
</head>
<body>
<div class="panel">
    Youtube URL: <input name='url'> <button id="add-btn">Start Converting</button>
</div>
<div id="root"></div>

<template id="yd-item-template">
    <style>
        :host {
            display: flex;
            flex-direction: column;
            
            box-shadow: 1px 1px 4px #452860;
            
            width: 300px;
            padding: 10px;
            margin: 5px;
        }
        
        header {
            padding: 3px 0 15px 0;
            font-size: 1.1em;
            font-weight: 600;
            text-align: left;
        }
        
        .progress {
            display: flex;
            flex-shrink: 0;
        }
        
        .sep {
            flex-grow: 1;
        }

        a[href=""] {
            display: none;
        }

        code {
            font-weight: 600;
            color: #90121d;
        }
    </style>
    <header></header>
	<div class="sep"></div>
    <div class="progress">
        <label></label>
        <div class="sep"></div>
        <progress max="100"></progress>
        <a href="" target="_blank">Download</a>
    </div>
    <code></code>
</template>

<script>
    class YdItemElement extends HTMLElement {
        constructor() {
            super();
            const template = document.querySelector('#yd-item-template').content;
            this.attachShadow({mode: 'open'}).appendChild(template.cloneNode(true));

            this.headerEl = this.shadowRoot.querySelector("header");
            this.statusEl = this.shadowRoot.querySelector("label");
            this.progressEl = this.shadowRoot.querySelector("progress");
            this.linkEl = this.shadowRoot.querySelector("a");
            this.errorEl = this.shadowRoot.querySelector("code");
            this.hasLink = false;
            this.progressEnabled = true;
        }

        static get observedAttributes() {
            return ['header', 'status', 'progress', 'link', 'error'];
        }

        attributeChangedCallback(name, oldValue, newValue) {
            switch (name) {
                case 'header':
                    this.headerEl.innerText = newValue;
                    break;
                case 'status':
                    this.statusEl.innerText = newValue;
                    break;
                case 'progress':
                    const progerss = Number(newValue);
                    this.progressEnabled = true;
                    
                    if (progerss > 0) {
                        this.progressEl.value = newValue;
                    } else if (progerss < 0) {
                        this.progressEnabled = false;
                    } else {
                        this.progressEl.removeAttribute("value");
                    }
                    break;
                case 'link':
                    this.hasLink = newValue !== "";

                    this.linkEl.href = newValue;
                    break;
                case 'error':
                    this.errorEl.innerText = newValue ? "Error: " + newValue : "";
                    break;
            }
            
            this.progressEl.style.display = this.hasLink 
                ? "none" 
                : this.progressEnabled 
                    ? "block" 
                    : "none";
        }
    }

    customElements.define('yd-item', YdItemElement);

    function startWebsocket() {
        const conn = new WebSocket("ws://{{.Host}}/ws");
        conn.onmessage = function (evt) {
            updateItem(JSON.parse(evt.data));
        };
    }

    function updateItem(data) {
        if (!data.url || !data.vid) {
            return;
        }
        const root = document.querySelector("#root");
        let item = document.getElementById(data.url);
        if (item === null) {
            item = document.createElement("yd-item");
            item.id = data.url;
            root.appendChild(item);
        }

        item.setAttribute('header', data.vid.title);
        item.setAttribute('status', data.status);
        item.setAttribute('progress', data.percent);
        item.setAttribute('link', data.downloadUrl);
        item.setAttribute('error', data.error);
    }
    
    function addNew(youtubeUrl) {
        fetch("//{{.Host}}/add", {
            method: "POST",
            body: JSON.stringify({url: youtubeUrl}),
            headers: {
                'Content-Type': 'application/json'
            }
        }); 
    }
    
    document.getElementById('add-btn').addEventListener('click', ev => {
        const inp = document.querySelector('input[name=url]');
        addNew(inp.value);
        inp.value = "";
    });

    fetch("/get").then(r => r.json()).then(r => {
        for (const d of r) {
            updateItem(d);
        }
    });

    startWebsocket();

</script>
</body>
</html>`

type vars struct {
	Host string
}

var indexTpl = template.Must(template.New("index").Parse(indexHtml))

func (d *dispatcher) IndexHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/html")
	err := indexTpl.Execute(writer, vars{request.Host})
	if err!= nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(err.Error()))
	}
}

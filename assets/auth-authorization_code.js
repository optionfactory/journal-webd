
function meta(k) {
    return document.head.querySelector(`meta[name=${k}]`).content
}

class Authorization {
    constructor(){
        let clientId = meta('clientId');
        let realmBaseUrl = meta('realmBaseUrl');
        this.flow = ful.AuthorizationCodeFlow.forKeycloak(clientId, realmBaseUrl, `${window.location.protocol}//${window.location.host}`);
    }

    async setup(){
        this.session = await this.flow.ensureLoggedIn();
        this.http = ful.HttpClient.builder().withInterceptors(this.session.interceptor()).build();
    }

    logout() {
        this.session.logout();
    }

    async authorizeWebSocket(socket){
        await this.session.refreshIf(2000);
        socket.send(JSON.stringify({"Authorization": this.session.token.access_token}));
    }
}

export const endpoints = {
    websocket: () => '__WSSCHEME' + '__APIROOT' + "/stream/ws",
    terms: () => '__APISCHEME' + '__APIROOT' + "/stream/terms"
}
export default endpoints;

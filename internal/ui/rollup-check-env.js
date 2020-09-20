export default function checkEnv(variables) {
    variables.forEach((v) => {
        if (!process.env[v]) {
            throw new Error("Missing environment variable: "+ v);
        }
    });
}

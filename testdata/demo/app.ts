const port = process.env.PORT;
const enabled = process.env.NEW_FEATURE_FLAG;
const exposed = import.meta.env.STRIPE_SECRET_KEY;
console.log(port, enabled, exposed);

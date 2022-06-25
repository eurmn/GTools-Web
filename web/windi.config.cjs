const { transform } = require('windicss/helpers')

module.exports = {
    content: ["./src/**/*.{html,js,jsx,tsx}", "./index.html"],
    theme: {
        extend: {
            fontFamily: {
                inter: ["\"InterVariable\"", "sans-serif"],
                quicksand: ["\"QuicksandVariable\"", "sans-serif"],
                ubuntu: ["\"UbuntuMonoVariable\"", "monospace"],
            }
        },
    },
    plugins: [
        require('windicss/plugin/aspect-ratio'),
        transform('daisyui'),
    ],
}
Emergency Shutdown Threshold Visualizer and Lucky Number Generator
-------------------
Also tells fortunes.

SPA
-------------------
A single page app (SPA) generator. Edit html, css, and js in separate files and have em all squished together into a single html file. Perfect for small, one-off SPAs with minimal setup.

* `src/index.html`: the template that the html, css, and js are injected into. You may want to edit this to include additional js or css deps.
* `src/app.html`: the html content for the SPA. This is injected into the document `<body>`
* `src/app.js`: javascript for the SPA. This is injected into an IIFE in a `<script>` tag at the end of the `<body>`
* `src/app.css`: css for this SPA. This is injected into a `<style>` tag in the document `<head>`

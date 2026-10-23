window.addEventListener("load", function () {
  SwaggerUIBundle({
    url: "/openapi.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    filter: true,
    docExpansion: "list",
    validatorUrl: null,
    supportedSubmitMethods: [],
    persistAuthorization: false
  });
});

# ingress-nginx 4.0.16

Generated by:

``` shell
helm template ingress-nginx ingress-nginx \
    --repo https://kubernetes.github.io/ingress-nginx \
    --version 4.0.16 \
    --namespace="ingress" \
    --no-hooks \
    > internal/functions/fixtures/render_helm_chart/ingress-nginx-4.0.16-no-hooks/fixture.yaml
```
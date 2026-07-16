# dtr-direc-tec-raf

Monorepo padre: tag-mule + tureparto

## Repositorios

- [tag_mule](https://github.com/doro098/tag_mule) (rama: master)
- [tureparto](https://github.com/doro098/tureparto) (rama: main)

## Instalación

Clona el repositorio e inicializa los submódulos:

```bash
git clone https://github.com/doro098/dtr-direc-tec-raf.git
cd dtr-direc-tec-raf
git submodule update --init --recursive
```

## Prerrequisitos

- Docker (Engine)
- docker-compose (o `docker compose` integrado)
- jq (opcional, para formatear JSON en la CLI)
- python3 (opcional, para pruebas rápidas)

## Cómo levantar el entorno (rápido)

Recomendado: usar Docker y docker-compose para levantar los servicios de forma reproducible.

1) Preparar la configuración de tag-mule

- Editá `tag_mule/config.yaml` (crealo si no existe) y asegurate de que el bloque para `tureparto` tenga un `callback_url` apuntando al receptor que usarás. Ejemplo mínimo:

```yaml
sources:
  tureparto:
    callback_url: http://cliente-tureparto:3001/webhook
    method: embedding
    provider: ollama
    model: nomic-embed-text
    max_tags: 3
    threshold: 0.6
    categories:
      - pedido
      - consulta
      - reclamo
      - direccion
      - horario
      - precio
      - producto
      - cancelacion
```

- Alternativa (si preferís ejecutar localmente sin YAML): en `tag_mule` existe `config.example.py`. Podés copiarlo a `config.py` y ajustar las variables de entorno (LLM_BASE_URL, LLM_MODEL, etc.).

2) Levantar servicios con docker-compose (desde la raíz del monorepo)

Si querés levantar solo el stack de tagging (ollama + tag-mule + cliente):

```bash
docker-compose -f tag_mule/docker-compose.yml up -d --build
```

(este compose monta `./tag-mule/config.yaml` en el contenedor de tag-mule — revisá que esté correcto antes de subir)

3) Verificar que tag-mule esté arriba

```bash
curl -s http://localhost:8080/api/v1/health | jq
```

Deberías ver algo como:

```json
{
  "status": "ok",
  "workers_active": 3,
  "jobs_pending": 0,
  "db_connected": true
}
```

4) Prueba rápida del flujo (callback)

Opción A — receptor temporal en tu host (debug local):

- Editá `tag_mule/config.yaml` y poné temporalmente el callback URL a `http://host.docker.internal:8089/webhook` (en Docker Desktop) o `http://localhost:8089/webhook` si tu red lo permite.

- Ejecutá un receptor temporal en el puerto 8089:

```bash
python3 - <<PY
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('content-length', 0))
        body = self.rfile.read(length).decode('utf-8')
        print('CALLBACK RECEIVED:', body)
        self.send_response(200)
        self.end_headers()

HTTPServer(('0.0.0.0', 8089), Handler).serve_forever()
PY
```

- Enviar un job a tag-mule:

```bash
curl -s -X POST http://localhost:8080/api/v1/enrich \
  -H 'Content-Type: application/json' \
  -d '{"source":"tureparto","item_id":"test-1","text":"Mensaje de prueba","existing_tags": []}' | jq
```

- Mirá la terminal del receptor: deberías ver el payload que tag-mule envía como callback.

Opción B — receptor dentro de Docker (cliente en compose):

- Asegurate que `tag_mule/config.yaml` tenga `callback_url: http://cliente-tureparto:3001/webhook`.
- Levantá el compose en `tag_mule/docker-compose.yml` (ver paso 2).
- Si existe el servicio `cliente-tureparto` en el compose, éste recibirá el callback y deberías ver logs con la petición.

## Comandos útiles

- Ver logs de tag-mule:
  - `docker-compose -f tag_mule/docker-compose.yml logs -f tag-mule`
- Listar contenedores: `docker ps`
- Parar y limpiar:

```bash
docker-compose -f tag_mule/docker-compose.yml down
# borrar imágenes locales y volúmenes (opcional)
docker-compose -f tag_mule/docker-compose.yml down --rmi local --volumes
```

## Notas y consejos

- En Mac/Windows, para que un contenedor alcance un servidor que corre en tu host usa `host.docker.internal` en la URL de callback.
- Asegurate que el `config.yaml` que montás en el contenedor (./tag-mule/config.yaml) coincida con la documentación y apunte al receptor correcto.
- `item_id` que envías a tag-mule corresponde internamente a `original_msg_id` en la DB de TuReparto.
- Si algo falla revisá los logs y el health endpoint; problemas típicos: puertos ocupados, URL del callback incorrecta, o modelos no cargados en Ollama.

---

Si querés, puedo: 
- añadir un `docker-compose.yml` en la raíz que orqueste ollama + tag-mule + cliente-tureparto, y
- añadir un script `tag_mule/scripts/test_integration.sh` para automatizar la prueba de callback.

Decime si querés que lo agregue y lo commiteo ahora.
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
- docker-compose o `docker compose` (integrado)
- jq (opcional, para formatear JSON en la CLI)
- python3 (opcional, para pruebas rápidas)

## Resumen: qué se levanta

El entorno de desarrollo mínimo recomendado levanta:
- Ollama (LLM runtime)
- Tag‑Mule (servicio Python que expone `/api/v1/enrich`)
- Cliente receptor `cliente-tureparto` (mock que recibe callbacks en `/webhook`)

Todos los servicios pueden orquestarse desde la raíz con `docker compose` y el `docker-compose.yml` raíz (o usando `tag_mule/docker-compose.yml` si preferís levantar solo ese subconjunto).

---

## Pasos para levantar el servidor (detallado y probado)

1) Preparar `tag_mule` config

- Editá (o creá) `tag_mule/config.yaml` y asegurate de que `sources.tureparto.callback_url` apunte a `http://cliente-tureparto:3001/webhook` si vas a usar el compose raíz. Ejemplo mínimo:

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

2) Levantar todo con docker-compose (recomendado)

- Desde la raíz del monorepo:

```bash
# Levanta ollama + tag-mule + cliente-tureparto
docker compose up -d --build
```

Si no tenés `docker compose` integrado, usá `docker-compose -f docker-compose.yml up -d --build`.

3) Alternativa: correr Ollama por separado (opcional)

Si no querés arrancar Ollama desde compose o lo tenés en otra máquina, podés lanzarlo en Docker con:

```bash
# correr Ollama en Docker (exponiendo puerto 11434)
docker run -d --name ollama -p 11434:11434 -v ollama_data:/root/.ollama ollama/ollama:latest
```

- Si arrancaste Ollama así, ajustá `tag_mule` para usar `http://host.docker.internal:11434` (Mac/Win) o `http://<IP_DEL_HOST>:11434` (Linux) como `OLLAMA_URL`.

4) Pre-cargar modelos en Ollama (recomendado antes de pruebas)

- Si Ollama corre en un contenedor llamado `ollama` (compose o docker run):

```bash
# con docker compose
docker compose exec -it ollama ollama pull qwen2.5:3b
docker compose exec -it ollama ollama pull nomic-embed-text

# ó con docker cli (si corristes docker run):
docker exec -it ollama ollama pull qwen2.5:3b
docker exec -it ollama ollama pull nomic-embed-text
```

Estos comandos descargan los modelos al volumen `ollama_data` y evitan tiempo de espera cuando tag-mule pide inferencias.

5) Verificar que tag-mule esté arriba

```bash
curl -s http://localhost:8080/api/v1/health | jq
```

Respuesta esperada (ejemplo):

```json
{
  "status": "ok",
  "workers_active": 3,
  "jobs_pending": 0,
  "db_connected": true
}
```

6) Probar el flujo completo (enviar job y recibir callback)

- Opción A — receptor dentro de Docker (compose raíz):
  - Asegurate que `tag_mule/config.yaml` use `http://cliente-tureparto:3001/webhook`.
  - Enviar job:

```bash
curl -s -X POST http://localhost:8080/api/v1/enrich \
  -H 'Content-Type: application/json' \
  -d '{"source":"tureparto","item_id":"test-1","text":"Mensaje de prueba","existing_tags": []}' | jq
```

  - Revisá los logs del receptor (cliente-tureparto):

```bash
docker compose logs -f cliente-tureparto
```

- Opción B — receptor temporal en el host (debug):
  - Cambiá temporalmente el callback a `http://host.docker.internal:8089/webhook` en `tag_mule/config.yaml` (Docker Desktop) y ejecutá el receptor Python en tu host (ver ejemplos en este README).

---

## Comandos útiles

- Ver contenedores: `docker ps`
- Ver logs en tiempo real:
  - `docker compose logs -f tag-mule`
  - `docker compose logs -f cliente-tureparto`
- Parar y limpiar:

```bash
docker compose down
# opcional: borrar imágenes locales y volúmenes
docker compose down --rmi local --volumes
```

## Requisitos de recursos y notas

- Ollama y los modelos pueden ser grandes: revisá RAM/CPU disponibles antes de descargar modelos pesados (p. ej. qwen2.5:3b). Para pruebas locales usa modelos más ligeros.
- Si Ollama está en el host y los contenedores deben llegar al host, usá `host.docker.internal` (Mac/Windows) o la IP del host en Linux.
- Asegurate de que las rutas `build:` en el `docker-compose.yml` sean correctas para tu estructura (ej.: `./tag_mule` y `./cliente_tureparto`).

---

Si querés, subo también los siguientes archivos al repo ahora mismo (commit directo):
- `docker-compose.yml` en la raíz (compose que orquesta ollama + tag-mule + cliente-tureparto)
- `cliente_tureparto/` (mock receiver: Dockerfile, app.py, requirements.txt)

Confirmame si querés que los cree y comitee ahora y lo hago. Si preferís pegarlos vos, ya están los contenidos arriba.
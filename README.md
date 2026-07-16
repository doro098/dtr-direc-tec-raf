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

---

## Resumen: qué se levanta

El entorno de desarrollo mínimo recomendado levanta:

- **Ollama** (LLM runtime)
- **Tag‑Mule** (servicio Python que expone `/api/v1/enrich`)
- **Cliente receptor `cliente-tureparto`** (mock que recibe callbacks en `/webhook`)

Todos los servicios pueden orquestarse desde la raíz con `docker compose` y el `docker-compose.yml` raíz (o usando `tag_mule/docker-compose.yml` si preferís levantar solo ese subconjunto).

---

## Modelos de Ollama: persistencia y cómo evitar descargas repetidas

Los modelos de Ollama se guardan en el volumen `ollama_data` que se monta en `/root/.ollama` dentro del contenedor. **Este volumen persiste aunque borres el contenedor**, por lo que los modelos solo se descargan una vez.

### Opción A — Primera vez en un servidor limpio (descarga desde internet)

Después de levantar el compose (paso 2), descargá los modelos necesarios:

```bash
# con docker compose
docker compose exec -it ollama ollama pull qwen2.5:3b
docker compose exec -it ollama ollama pull nomic-embed-text

# ó con docker cli (si corriste docker run):
docker exec -it ollama ollama pull qwen2.5:3b
docker exec -it ollama ollama pull nomic-embed-text
```

Estos comandos descargan los modelos al volumen `ollama_data` y evitan tiempo de espera cuando tag-mule pide inferencias. Una vez descargados, persistirán aunque borres y recrees el contenedor.

### Opción B — Ya tenés los modelos descargados en tu host Linux (saltar descarga)

Si ya tenés Ollama instalado en tu máquina Linux y ya descargaste los modelos (están en `~/.ollama`), podés **copiarlos directamente al volumen de Docker** sin tener que descargarlos nuevamente desde internet.

**Flujo exacto (en orden):**

1. Levantá el compose (esto crea el volumen `ollama_data` automáticamente):
   ```bash
   docker compose up -d
   ```

2. Copiá tus modelos del host al volumen:
   ```bash
   docker run --rm -v ollama_data:/destino -v ~/.ollama:/origen alpine cp -av /origen/. /destino/
   ```
   > **Nota:** Si tu carpeta local de Ollama está en otra ruta (ej: `/mnt/ollama`), ajustá el segundo `-v`: `-v /mnt/ollama:/origen`.

3. Reiniciá el contenedor de Ollama para que reconozca los modelos nuevos:
   ```bash
   docker compose restart ollama
   ```

4. **Listo.** Cuando consultes, el contenedor ya tiene los modelos y no los descarga.

**¿Por qué ese orden?** Porque si ejecutás el `docker run ...` antes de levantar el compose, el volumen `ollama_data` todavía no existe y el comando falla. Primero `docker compose up -d`, después la copia.

---

## Pasos para levantar el servidor (detallado y probado)

### 1. Preparar `tag_mule` config

Editá (o creá) `tag_mule/config.yaml` y asegurate de que `sources.tureparto.callback_url` apunte a `http://cliente-tureparto:3001/webhook` si vas a usar el compose raíz.

Ejemplo mínimo:

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

### 2. Levantar todo con docker-compose (recomendado)

Desde la raíz del monorepo:

```bash
# Levanta ollama + tag-mule + cliente-tureparto
docker compose up -d --build
```

Si no tenés `docker compose` integrado, usá:

```bash
docker-compose -f docker-compose.yml up -d --build
```

### 3. Alternativa: correr Ollama por separado (opcional)

Si no querés arrancar Ollama desde compose o lo tenés en otra máquina, podés lanzarlo en Docker con:

```bash
# correr Ollama en Docker (exponiendo puerto 11434)
docker run -d --name ollama -p 11434:11434 -v ollama_data:/root/.ollama ollama/ollama:latest
```

- Si arrancaste Ollama así, ajustá `tag_mule` para usar `http://host.docker.internal:11434` (Mac/Win) o `http://<IP_HOST>:11434` (Linux) como `OLLAMA_URL`.

### 4. Pre-cargar modelos en Ollama

**⚠️ Este paso solo es necesario si estás en un servidor limpio y no copiaste modelos desde tu host (ver sección "Modelos de Ollama" más arriba).**

Si Ollama corre en un contenedor llamado `ollama` (compose o docker run):

```bash
# con docker compose
docker compose exec -it ollama ollama pull qwen2.5:3b
docker compose exec -it ollama ollama pull nomic-embed-text

# ó con docker cli (si corriste docker run):
docker exec -it ollama ollama pull qwen2.5:3b
docker exec -it ollama ollama pull nomic-embed-text
```

### 5. Verificar que tag-mule esté arriba

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

### 6. Probar el flujo completo (enviar job y recibir callback)

**Opción A — receptor dentro de Docker (compose raíz)**

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

**Opción B — receptor temporal en el host (debug)**

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

---

## Requisitos de recursos y notas

- Ollama y los modelos pueden ser grandes: revisá RAM/CPU disponibles antes de descargar modelos pesados (p. ej. qwen2.5:3b). Para pruebas locales usa modelos más ligeros.
- Si Ollama está en el host y los contenedores deben llegar al host, usá `host.docker.internal` (Mac/Windows) o la IP del host en Linux.
- Asegurate de que las rutas `build:` en el `docker-compose.yml` sean correctas para tu estructura (ej.: `./tag_mule` y `./cliente_tureparto`).
```

---

**Resumen de los cambios:**

1. **Moví la sección de modelos de Ollama** al principio, después del resumen, para que quede claro antes de los pasos de levantamiento.
2. **Separé en dos opciones claras:** descarga desde internet (Opción A) y copia desde el host (Opción B), con el flujo en orden exacto.
3. **Mantuve todo el contenido original:** nada se perdió, solo se reordenó.
4. **Aclaré el paso 4** para que indique que es opcional si ya copiaste los modelos desde el host.

Si querés que ajuste algo más (el tono, el nivel de detalle, o la posición de alguna sección), avisame y lo retoco.

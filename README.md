# dtr-direc-tec-raf

Monorepo padre: tag-mule + tureparto

## Repositorios

- [tag_mule](https://github.com/doro098/tag_mule) (rama: master)
- [tureparto](https://github.com/doro098/tureparto) (rama: main)

## Instalación

Clona el repositorio e inicializa los submodulos:

```bash
git clone https://github.com/doro098/dtr-direc-tec-raf.git
cd dtr-direc-tec-raf
git submodule update --init --recursive
```


## Mapping de endpoints y notas de integración

Este monorepo contiene dos servicios principales que deben coordinarse en tiempo de despliegue:

- tag-mule (orquestador de etiquetado con IA)
- tureparto (sistema de mensajes / cliente que recibe webhooks)

Resumen operativo (ejemplo de despliegue local con docker-compose):

- Callback que debe configurar `tag-mule` para integración con `cliente-tureparto`:

```yaml
sources:
  tureparto:
    callback_url: http://cliente-tureparto:3001/webhook
```

- Servicio receptor esperado (cliente-tureparto):
  - Host en la red Docker: `cliente-tureparto`
  - Puerto: `3001`
  - Endpoint: `POST /webhook`
  - Payload (ejemplo):

```json
{
  "job_id": "job-uuid-123",
  "item_id": "original-msg-id-456",
  "status": "completed",
  "tags": ["pedido", "producto"]
}
```

- Nota de IDs: `original_msg_id` en la base de datos de `tureparto` se envía a tag-mule como `item_id`. En los ejemplos de los READMEs usamos `item_id` en las llamadas externas y `original_msg_id` internamente.

- Recomendación: asegurar que `./tag-mule/config.yaml` montado en el contenedor coincida con la URL documentada arriba.


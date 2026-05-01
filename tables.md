
---

# Tablas principales de la base de datos

## 1. `categories`

**Propósito:**
Agrupa los documentos a nivel general.

| Campo         | Tipo sugerido        |    Restricciones | Descripción                                        |
| ------------- | -------------------- | ---------------: | -------------------------------------------------- |
| `id`          | string               |    PK, requerido | Identificador único de la categoría                |
| `name`        | string               | requerido, único | Nombre de la categoría                             |
| `description` | string               |         opcional | Descripción breve de la categoría                  |
| `tags`        | json / array[string] |         opcional | Etiquetas base para apoyo inicial de clasificación |
| `color`       | string               |         opcional | Color para representación en interfaz              |
| `icon`        | string               |         opcional | Ícono asociado a la categoría                      |
| `created`     | datetime             |       automático | Fecha de creación                                  |
| `updated`     | datetime             |       automático | Fecha de actualización                             |

**Relaciones:**

* `categories.id` → `subcategories.category_id`
* `categories.id` → `documents.category_id`

---

## 2. `subcategories`

**Propósito:**
Subdivide una categoría principal. Cada categoría debe tener al menos una subcategoría llamada `default`.

| Campo         | Tipo sugerido        |              Restricciones | Descripción                              |
| ------------- | -------------------- | -------------------------: | ---------------------------------------- |
| `id`          | string               |              PK, requerido | Identificador único de la subcategoría   |
| `category_id` | relation / string    |              FK, requerido | Categoría a la que pertenece             |
| `name`        | string               |                  requerido | Nombre de la subcategoría                |
| `description` | string               |                   opcional | Descripción breve                        |
| `model_name`  | string               |                  requerido | Nombre del modelo asociado en backend    |
| `tags`        | json / array[string] |                   opcional | Etiquetas iniciales de apoyo             |
| `is_default`  | bool                 | requerido, default `false` | Indica si es la subcategoría por defecto |
| `created`     | datetime             |                 automático | Fecha de creación                        |
| `updated`     | datetime             |                 automático | Fecha de actualización                   |

**Reglas sugeridas:**

* Una categoría debe tener exactamente una subcategoría con `is_default = true`.
* `name` puede repetirse entre categorías distintas, pero no dentro de la misma categoría.

**Relaciones:**

* `subcategories.category_id` → `categories.id`
* `subcategories.id` → `documents.subcategory_id`

---

## 3. `documents`

**Propósito:**
Es la entidad central del sistema. Almacena el archivo, el texto extraído y su organización dentro de Fileoteca.

| Campo            | Tipo sugerido     |                Restricciones | Descripción                                                      |
| ---------------- | ----------------- | ---------------------------: | ---------------------------------------------------------------- |
| `id`             | string            |                PK, requerido | Identificador único del documento                                |
| `name`           | string            |                    requerido | Nombre visible dentro del sistema                                |
| `file_name`      | string            |                    requerido | Nombre original del archivo                                      |
| `file_ext`       | string            |                    requerido | Extensión del archivo (`pdf`, `docx`, `xlsx`)                    |
| `file_size`      | number            |                    requerido | Tamaño del archivo en bytes                                      |
| `path`           | string            |                    requerido | Ruta local del archivo                                           |
| `hash`           | string            |   opcional, idealmente único | Hash del archivo para detectar duplicados                        |
| `ocr_txt`        | text              |                     opcional | Texto extraído del documento                                     |
| `metadata`       | json              |                     opcional | Metadatos asociados al documento                                 |
| `category_id`    | relation / string |                FK, requerido | Categoría asignada                                               |
| `subcategory_id` | relation / string |                FK, requerido | Subcategoría asignada                                            |
| `status`         | string            | requerido, default `pending` | Estado del documento (`pending`, `processed`, `error`)           |
| `source_type`    | string            |                     opcional | Origen de ingreso (`context_menu`, `manual_upload`, `drag_drop`) |
| `notes`          | text              |                     opcional | Notas manuales del usuario                                       |
| `created`        | datetime          |                   automático | Fecha de creación                                                |
| `updated`        | datetime          |                   automático | Fecha de actualización                                           |

**Relaciones:**

* `documents.category_id` → `categories.id`
* `documents.subcategory_id` → `subcategories.id`
* `documents.id` → `document_tags.document_id`
* `documents.id` → `import_items.document_id`

---

# Tablas auxiliares recomendadas

## 4. `document_tags`

**Propósito:**
Permite asignar etiquetas manuales a documentos, independientes de categorías y subcategorías.

| Campo         | Tipo sugerido     | Restricciones | Descripción         |
| ------------- | ----------------- | ------------: | ------------------- |
| `id`          | string            | PK, requerido | Identificador único |
| `document_id` | relation / string | FK, requerido | Documento asociado  |
| `tag`         | string            |     requerido | Etiqueta manual     |
| `created`     | datetime          |    automático | Fecha de creación   |

**Reglas sugeridas:**

* Evitar duplicados de `tag` para un mismo `document_id`.

**Relaciones:**

* `document_tags.document_id` → `documents.id`

---

## 5. `imports`

**Propósito:**
Registra procesos de importación de archivos a Fileoteca.

| Campo             | Tipo sugerido |                Restricciones | Descripción                                                         |
| ----------------- | ------------- | ---------------------------: | ------------------------------------------------------------------- |
| `id`              | string        |                PK, requerido | Identificador único de la importación                               |
| `source_path`     | string        |                     opcional | Ruta origen desde donde se inició la importación                    |
| `source_type`     | string        |                    requerido | Tipo de origen (`context_menu`, `manual_upload`, `drag_drop`)       |
| `total_files`     | number        |       requerido, default `0` | Total de archivos incluidos                                         |
| `processed_files` | number        |       requerido, default `0` | Total de archivos procesados                                        |
| `status`          | string        | requerido, default `pending` | Estado de la importación (`pending`, `processing`, `done`, `error`) |
| `error_message`   | text          |                     opcional | Mensaje de error general                                            |
| `created`         | datetime      |                   automático | Fecha de creación                                                   |
| `updated`         | datetime      |                   automático | Fecha de actualización                                              |

**Relaciones:**

* `imports.id` → `import_items.import_id`

---

## 6. `import_items`

**Propósito:**
Detalla cada archivo perteneciente a una importación.

| Campo           | Tipo sugerido     |                Restricciones | Descripción                                         |
| --------------- | ----------------- | ---------------------------: | --------------------------------------------------- |
| `id`            | string            |                PK, requerido | Identificador único                                 |
| `import_id`     | relation / string |                FK, requerido | Importación a la que pertenece                      |
| `document_id`   | relation / string |                 FK, opcional | Documento creado, si aplica                         |
| `original_path` | string            |                    requerido | Ruta original del archivo                           |
| `status`        | string            | requerido, default `pending` | Estado (`pending`, `processed`, `skipped`, `error`) |
| `error_message` | text              |                     opcional | Error individual del archivo                        |
| `created`       | datetime          |                   automático | Fecha de creación                                   |
| `updated`       | datetime          |                   automático | Fecha de actualización                              |

**Relaciones:**

* `import_items.import_id` → `imports.id`
* `import_items.document_id` → `documents.id`

---

## 7. `app_settings`

**Propósito:**
Guarda configuración persistente de la aplicación local.

| Campo         | Tipo sugerido |    Restricciones | Descripción            |
| ------------- | ------------- | ---------------: | ---------------------- |
| `id`          | string        |    PK, requerido | Identificador único    |
| `key`         | string        | requerido, único | Clave de configuración |
| `value`       | json / string |        requerido | Valor asociado         |
| `description` | string        |         opcional | Descripción del ajuste |
| `updated`     | datetime      |       automático | Fecha de actualización |

**Ejemplos de claves:**

* `library_root_path`
* `default_export_path`
* `ocr_enabled`
* `auto_classify_enabled`
* `preferred_ocr_engine`

---

# Tablas opcionales

Estas no son obligatorias para el MVP, pero sí pueden mencionarse como expansión natural del sistema.

## 8. `folders`

**Propósito:**
Permite organización manual complementaria en estructura tipo biblioteca.

| Campo         | Tipo sugerido     | Restricciones | Descripción                  |
| ------------- | ----------------- | ------------: | ---------------------------- |
| `id`          | string            | PK, requerido | Identificador único          |
| `name`        | string            |     requerido | Nombre de la carpeta         |
| `description` | string            |      opcional | Descripción                  |
| `parent_id`   | relation / string |  FK, opcional | Carpeta padre para jerarquía |
| `created`     | datetime          |    automático | Fecha de creación            |
| `updated`     | datetime          |    automático | Fecha de actualización       |

**Relaciones:**

* `folders.parent_id` → `folders.id`
* `folders.id` ↔ `documents.id` mediante `document_folders`

---

## 9. `document_folders`

**Propósito:**
Relaciona documentos con carpetas.

| Campo         | Tipo sugerido     | Restricciones | Descripción         |
| ------------- | ----------------- | ------------: | ------------------- |
| `id`          | string            | PK, requerido | Identificador único |
| `document_id` | relation / string | FK, requerido | Documento asociado  |
| `folder_id`   | relation / string | FK, requerido | Carpeta asociada    |
| `created`     | datetime          |    automático | Fecha de creación   |

**Relaciones:**

* `document_folders.document_id` → `documents.id`
* `document_folders.folder_id` → `folders.id`

---

## 10. `saved_filters`

**Propósito:**
Permite guardar vistas o filtros frecuentes dentro de la aplicación.

| Campo           | Tipo sugerido | Restricciones | Descripción                          |
| --------------- | ------------- | ------------: | ------------------------------------ |
| `id`            | string        | PK, requerido | Identificador único                  |
| `name`          | string        |     requerido | Nombre del filtro guardado           |
| `filter_config` | json          |     requerido | Configuración serializada del filtro |
| `created`       | datetime      |    automático | Fecha de creación                    |
| `updated`       | datetime      |    automático | Fecha de actualización               |

---

## 11. `document_links`

**Propósito:**
Permite relacionar documentos entre sí.

| Campo                | Tipo sugerido     | Restricciones | Descripción                                             |
| -------------------- | ----------------- | ------------: | ------------------------------------------------------- |
| `id`                 | string            | PK, requerido | Identificador único                                     |
| `document_id`        | relation / string | FK, requerido | Documento origen                                        |
| `linked_document_id` | relation / string | FK, requerido | Documento relacionado                                   |
| `relation_type`      | string            |     requerido | Tipo de relación (`related`, `attachment`, `reference`) |
| `created`            | datetime          |    automático | Fecha de creación                                       |

**Relaciones:**

* `document_links.document_id` → `documents.id`
* `document_links.linked_document_id` → `documents.id`

---

# Relaciones generales del sistema

## Relaciones obligatorias

```md
categories (1) ---- (N) subcategories
categories (1) ---- (N) documents
subcategories (1) ---- (N) documents
documents (1) ---- (N) document_tags
imports (1) ---- (N) import_items
documents (1) ---- (N) import_items
```

## Relaciones opcionales

```md
folders (1) ---- (N) folders
documents (N) ---- (N) folders   [mediante document_folders]
documents (N) ---- (N) documents [mediante document_links]
```

---

# Resumen recomendado para el documento

## Núcleo del MVP

```md
- categories
- subcategories
- documents
- document_tags
- imports
- import_items
- app_settings
```

## Extensiones opcionales

```md
- folders
- document_folders
- saved_filters
- document_links
```

---

# Diseño lógico resumido

```md
Category
 └── SubCategory
      └── Document
           └── DocumentTags

Import
 └── ImportItems
      └── Document

AppSettings
```

---

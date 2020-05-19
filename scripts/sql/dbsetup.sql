CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION add_fhir_endpoint_info_history() RETURNS TRIGGER AS $fhir_endpoints_info_history$
    BEGIN
        --
        -- Create a row in fhir_endpoints_info_history to reflect the operation performed on fhir_endpoints_info,
        -- make use of the special variable TG_OP to work out the operation.
        --
        IF (TG_OP = 'DELETE') THEN
            INSERT INTO fhir_endpoints_info_history SELECT 'D', now(), user, OLD.*;
            RETURN OLD;
        ELSIF (TG_OP = 'UPDATE') THEN
            INSERT INTO fhir_endpoints_info_history SELECT 'U', now(), user, NEW.*;
            RETURN NEW;
        ELSIF (TG_OP = 'INSERT') THEN
            INSERT INTO fhir_endpoints_info_history SELECT 'I', now(), user, NEW.*;
            RETURN NEW;
        END IF;
        RETURN NULL; -- result is ignored since this is an AFTER trigger
    END;
$fhir_endpoints_info_history$ LANGUAGE plpgsql;

CREATE TABLE npi_organizations (
    id               SERIAL PRIMARY KEY,
    npi_id			     VARCHAR(500) UNIQUE,
    name             VARCHAR(500),
    secondary_name   VARCHAR(500),
    location         JSONB,
    taxonomy 		     VARCHAR(500), -- Taxonomy code mapping: http://www.wpc-edi.com/reference/codelists/healthcare/health-care-provider-taxonomy-code-set/
    normalized_name      VARCHAR(500),
    normalized_secondary_name   VARCHAR(500),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE npi_contacts (
    id               SERIAL PRIMARY KEY,
    npi_id			     VARCHAR(500),
	endpoint_type   VARCHAR(500),
	endpoint_type_description   VARCHAR(500),
	endpoint   VARCHAR(500),
    valid_url BOOLEAN,
	affiliation   VARCHAR(500),
	endpoint_description   VARCHAR(500),
	affiliation_legal_business_name   VARCHAR(500),
	use_code   VARCHAR(500),
	use_description   VARCHAR(500),
	other_use_description   VARCHAR(500),
	content_type   VARCHAR(500),
	content_description   VARCHAR(500),
	other_content_description   VARCHAR(500),
    location                JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vendors (
    id                      SERIAL PRIMARY KEY,
    name                    VARCHAR(500) UNIQUE,
    developer_code          VARCHAR(500) UNIQUE,
    url                     VARCHAR(500),
    location                JSONB,
    status                  VARCHAR(500),
    last_modified_in_chpl   TIMESTAMPTZ,
    chpl_id                 INTEGER UNIQUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE healthit_products (
    id                      SERIAL PRIMARY KEY,
    name                    VARCHAR(500),
    version                 VARCHAR(500),
    vendor_id               INT REFERENCES vendors(id) ON DELETE CASCADE,
    location                JSONB,
    authorization_standard  VARCHAR(500),
    api_syntax              VARCHAR(500),
    api_url                 VARCHAR(500),
    certification_criteria  JSONB,
    certification_status    VARCHAR(500),
    certification_date      DATE,
    certification_edition   VARCHAR(500),
    last_modified_in_chpl   DATE,
    chpl_id                 VARCHAR(500),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT healthit_product_info UNIQUE(name, version)
);

CREATE TABLE fhir_endpoints (
    id                      SERIAL PRIMARY KEY,
    url                     VARCHAR(500),
    organization_name       VARCHAR(500),
    list_source             VARCHAR(500),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fhir_endpoints_unique UNIQUE(url, list_source)
);

CREATE TABLE fhir_endpoints_info (
    id                      SERIAL PRIMARY KEY,
    healthit_product_id     INT REFERENCES healthit_products(id) ON DELETE SET NULL,
    vendor_id               INT REFERENCES vendors(id) ON DELETE SET NULL, 
    url                     VARCHAR(500) UNIQUE,
    tls_version             VARCHAR(500),
    mime_types              VARCHAR(500)[],
    http_response           INTEGER,
    errors                  VARCHAR(500),
    capability_statement    JSONB,
    validation              JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE fhir_endpoints_info_history (
    operation               CHAR(1) NOT NULL,
    entered_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id                 VARCHAR(500),
    id                      INT, -- should link to fhir_endpoints_info(id). not using 'reference' because if the original is deleted, we still want the historical copies to remain and keep the ID so they can be linked to one another.
    healthit_product_id     INT, -- should link to healthit_product(id). not using 'reference' because if the referenced product is deleted, we still want the historical copies to retain the ID.
    vendor_id               INT,  -- should link to vendor_id(id). not using 'reference' because if the referenced vendor is deleted, we still want the historical copies to retain the ID.
    url                     VARCHAR(500),
    tls_version             VARCHAR(500),
    mime_types              VARCHAR(500)[],
    http_response           INTEGER,
    errors                  VARCHAR(500),
    capability_statement    JSONB,
    validation              JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE endpoint_organization (
    endpoint_id INT REFERENCES fhir_endpoints (id) ON DELETE CASCADE,
    organization_id INT REFERENCES npi_organizations (id) ON DELETE CASCADE,
    confidence NUMERIC (5, 3),
    CONSTRAINT endpoint_org PRIMARY KEY (endpoint_id, organization_id)
);

CREATE INDEX fhir_endpoint_url_index ON fhir_endpoints (url);

CREATE TRIGGER set_timestamp_fhir_endpoints
BEFORE UPDATE ON fhir_endpoints
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp_npi_organization
BEFORE UPDATE ON npi_organizations
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp_vendors
BEFORE UPDATE ON vendors
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp_healthit_products
BEFORE UPDATE ON healthit_products
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp_fhir_endpoints_info
BEFORE UPDATE ON fhir_endpoints_info
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- captures history for the fhir_endpoint_info table
CREATE TRIGGER add_fhir_endpoint_info_history_trigger
AFTER INSERT OR UPDATE OR DELETE on fhir_endpoints_info
FOR EACH ROW
EXECUTE PROCEDURE add_fhir_endpoint_info_history();


CREATE or REPLACE VIEW org_mapping AS
SELECT endpts.url, vendors.name, endpts.organization_name AS endpoint_name, orgs.name AS ORGANIZATION_NAME, orgs.secondary_name AS ORGANIZATION_SECONDARY_NAME, orgs.taxonomy, orgs.Location->>'state' AS STATE, orgs.Location->>'zipcode' AS ZIPCODE, links.confidence AS MATCH_SCORE
FROM endpoint_organization AS links
LEFT JOIN fhir_endpoints AS endpts ON links.endpoint_id = endpts.id
LEFT JOIN npi_organizations AS orgs ON links.organization_id = orgs.id
LEFT JOIN fhir_endpoints_info AS endpts_info ON endpts.url = endpts_info.url
LEFT JOIN vendors ON endpts_info.vendor_id = vendors.id;

CREATE or REPLACE VIEW endpoint_export AS
SELECT endpts.url, vendors.name as vendor_name, endpts.organization_name AS endpoint_name, endpts_info.tls_version, endpts_info.mime_types, endpts_info.http_response, endpts_info.capability_statement->>'fhirVersion' AS FHIR_VERSION, endpts_info.capability_statement->>'publisher' AS PUBLISHER, endpts_info.capability_statement->'software'->'name' AS SOFTWARE_NAME, endpts_info.capability_statement->'software'->'version' AS SOFTWARE_VERSION, endpts_info.capability_statement->'software'->'releaseDate' AS SOFTWARE_RELEASEDATE, orgs.name AS ORGANIZATION_NAME, orgs.secondary_name AS ORGANIZATION_SECONDARY_NAME, orgs.taxonomy, orgs.Location->>'state' AS STATE, orgs.Location->>'zipcode' AS ZIPCODE, links.confidence AS MATCH_SCORE
FROM endpoint_organization AS links
RIGHT JOIN fhir_endpoints AS endpts ON links.endpoint_id = endpts.id
LEFT JOIN npi_organizations AS orgs ON links.organization_id = orgs.id
LEFT JOIN fhir_endpoints_info AS endpts_info ON endpts.url = endpts_info.url
LEFT JOIN vendors ON endpts_info.vendor_id = vendors.id;

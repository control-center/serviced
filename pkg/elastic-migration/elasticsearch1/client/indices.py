from .utils import NamespacedClient, query_params, _make_path, SKIP_IN_PATH
from ..exceptions import NotFoundError

class IndicesClient(NamespacedClient):
    @query_params('analyzer', 'char_filters', 'field', 'filters', 'format',
        'prefer_local', 'text', 'tokenizer')
    def analyze(self, index=None, body=None, params=None):
        """
        Perform the analysis process on a text and return the tokens breakdown of the text.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-analyze.html>`_

        :arg index: The name of the index to scope the operation
        :arg body: The text on which the analysis should be performed
        :arg analyzer: The name of the analyzer to use
        :arg char_filters: A comma-separated list of character filters to use
            for the analysis
        :arg field: Use the analyzer configured for this field (instead of
            passing the analyzer name)
        :arg filters: A comma-separated list of filters to use for the analysis
        :arg format: Format of the output, default 'detailed', valid choices
            are: 'detailed', 'text'
        :arg prefer_local: With `true`, specify that a local shard should be
            used if available, with `false`, use a random shard (default: true)
        :arg text: The text on which the analysis should be performed (when
            request body is not used)
        :arg tokenizer: The name of the tokenizer to use for the analysis
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_analyze'), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'force',
        'ignore_unavailable', 'operation_threading')
    def refresh(self, index=None, params=None):
        """
        Explicitly refresh one or more index, making all operations performed
        since the last refresh available for search.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-refresh.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg force: Force a refresh even if not required, default False
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg operation_threading: TODO: ?
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_refresh'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'force',
        'ignore_unavailable', 'wait_if_ongoing')
    def flush(self, index=None, params=None):
        """
        Explicitly flush one or more indices.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-flush.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string for all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg force: Whether a flush should be forced even if it is not
            necessarily needed ie. if no changes will be committed to the index.
            This is useful if transaction log IDs should be incremented even if
            no uncommitted changes are present. (This setting can be considered
            as internal)
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg wait_if_ongoing: If set to true the flush operation will block
            until the flush can be executed if another flush operation is
            already executing. The default is false and will cause an exception
            to be thrown on the shard level if another flush operation is
            already running.
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_flush'), params=params)
        return data

    @query_params('master_timeout', 'timeout')
    def create(self, index, body=None, params=None):
        """
        Create an index in Elasticsearch.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html>`_

        :arg index: The name of the index
        :arg body: The configuration for the index (`settings` and `mappings`)
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        _, data = self.transport.perform_request('PUT', _make_path(index),
            params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'flat_settings',
        'human', 'ignore_unavailable', 'local')
    def get(self, index, feature=None, params=None):
        """
        The get index API allows to retrieve information about one or more indexes.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-get-index.html>`_

        :arg index: A comma-separated list of index names
        :arg feature: A comma-separated list of features
        :arg allow_no_indices: Ignore if a wildcard expression resolves to no
            concrete indices (default: false)
        :arg expand_wildcards: Whether wildcard expressions should get expanded
            to open or closed indices (default: open), default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg flat_settings: Return settings in flat format (default: false)
        :arg human: Whether to return version and creation date values in human-
            readable format., default False
        :arg ignore_unavailable: Ignore unavailable indexes (default: false)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        _, data = self.transport.perform_request('GET', _make_path(index,
            feature), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'master_timeout', 'timeout')
    def open(self, index, params=None):
        """
        Open a closed index to make it available for search.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-open-close.html>`_

        :arg index: The name of the index
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'closed', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_open'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'master_timeout', 'timeout')
    def close(self, index, params=None):
        """
        Close an index to remove it's overhead from the cluster. Closed index
        is blocked for read/write operations.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-open-close.html>`_

        :arg index: The name of the index
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_close'), params=params)
        return data

    @query_params('master_timeout', 'timeout')
    def delete(self, index, params=None):
        """
        Delete an index in Elasticsearch
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-delete-index.html>`_

        :arg index: A comma-separated list of indices to delete; use `_all` or
            `*` string to delete all indices
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        _, data = self.transport.perform_request('DELETE', _make_path(index),
            params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def exists(self, index, params=None):
        """
        Return a boolean indicating whether given index exists.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-exists.html>`_

        :arg index: A comma-separated list of indices to check
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        try:
            self.transport.perform_request('HEAD', _make_path(index),
                params=params)
        except NotFoundError:
            return False
        return True

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def exists_type(self, index, doc_type, params=None):
        """
        Check if a type/types exists in an index/indices.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-types-exists.html>`_

        :arg index: A comma-separated list of index names; use `_all` to check
            the types across all indices
        :arg doc_type: A comma-separated list of document types to check
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        for param in (index, doc_type):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        try:
            self.transport.perform_request('HEAD', _make_path(index, doc_type),
                params=params)
        except NotFoundError:
            return False
        return True

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_conflicts',
        'ignore_unavailable', 'master_timeout', 'timeout')
    def put_mapping(self, doc_type, body, index=None, params=None):
        """
        Register specific mapping definition for a specific type.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-put-mapping.html>`_

        :arg doc_type: The name of the document type
        :arg body: The mapping definition
        :arg index: A comma-separated list of index names the mapping should be
            added to (supports wildcards); use `_all` or omit to add the mapping
            on all indices.
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_conflicts: Specify whether to ignore conflicts while
            updating the mapping (default: false)
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        for param in (doc_type, body):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('PUT', _make_path(index,
            '_mapping', doc_type), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def get_mapping(self, index=None, doc_type=None, params=None):
        """
        Retrieve mapping definition of index or index/type.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-get-mapping.html>`_

        :arg index: A comma-separated list of index names
        :arg doc_type: A comma-separated list of document types
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_mapping', doc_type), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'include_defaults', 'local')
    def get_field_mapping(self, field, index=None, doc_type=None, params=None):
        """
        Retrieve mapping definition of a specific field.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-get-field-mapping.html>`_

        :arg field: A comma-separated list of fields
        :arg index: A comma-separated list of index names
        :arg doc_type: A comma-separated list of document types
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg include_defaults: Whether the default mapping values should be
            returned as well
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        if field in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'field'.")
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_mapping', doc_type, 'field', field), params=params)
        return data

    @query_params('master_timeout')
    def delete_mapping(self, index, doc_type, params=None):
        """
        Delete a mapping (type) along with its data.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-delete-mapping.html>`_

        :arg index: A comma-separated list of index names (supports wildcard);
            use `_all` for all indices
        :arg doc_type: A comma-separated list of document types to delete
            (supports wildcards); use `_all` to delete all document types in the
            specified indices.
        :arg master_timeout: Specify timeout for connection to master
        """
        for param in (index, doc_type):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('DELETE', _make_path(index, '_mapping', doc_type),
            params=params)
        return data
    @query_params('master_timeout', 'timeout')
    def put_alias(self, index, name, body=None, params=None):
        """
        Create an alias for a specific index/indices.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg index: A comma-separated list of index names the alias should point
            to (supports wildcards); use `_all` to perform the operation on all
            indices.
        :arg name: The name of the alias to be created or updated
        :arg body: The settings for the alias, such as `routing` or `filter`
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit timeout for the operation
        """
        for param in (index, name):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('PUT', _make_path(index,
            '_alias', name), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def exists_alias(self, index=None, name=None, params=None):
        """
        Return a boolean indicating whether given alias exists.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg index: A comma-separated list of index names to filter aliases
        :arg name: A comma-separated list of alias names to return
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default ['open', 'closed'],
            valid choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        try:
            self.transport.perform_request('HEAD', _make_path(index, '_alias',
                name), params=params)
        except NotFoundError:
            return False
        return True

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def get_alias(self, index=None, name=None, params=None):
        """
        Retrieve a specified alias.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg index: A comma-separated list of index names to filter aliases
        :arg name: A comma-separated list of alias names to return
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_alias', name), params=params)
        return data

    @query_params('local', 'timeout')
    def get_aliases(self, index=None, name=None, params=None):
        """
        Retrieve specified aliases
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg index: A comma-separated list of index names to filter aliases
        :arg name: A comma-separated list of alias names to filter
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        :arg timeout: Explicit operation timeout
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_aliases', name), params=params)
        return data

    @query_params('master_timeout', 'timeout')
    def update_aliases(self, body, params=None):
        """
        Update specified aliases.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg body: The definition of `actions` to perform
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Request timeout
        """
        if body in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'body'.")
        _, data = self.transport.perform_request('POST', '/_aliases',
            params=params, body=body)
        return data

    @query_params('master_timeout', 'timeout')
    def delete_alias(self, index, name, params=None):
        """
        Delete specific alias.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-aliases.html>`_

        :arg index: A comma-separated list of index names (supports wildcards);
            use `_all` for all indices
        :arg name: A comma-separated list of aliases to delete (supports
            wildcards); use `_all` to delete all aliases for the specified
            indices.
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit timeout for the operation
        """
        for param in (index, name):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('DELETE', _make_path(index,
            '_alias', name), params=params)
        return data

    @query_params('create', 'flat_settings', 'master_timeout', 'order',
        'timeout')
    def put_template(self, name, body, params=None):
        """
        Create an index template that will automatically be applied to new
        indices created.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates.html>`_

        :arg name: The name of the template
        :arg body: The template definition
        :arg create: Whether the index template should only be added if new or
            can also replace an existing one, default False
        :arg flat_settings: Return settings in flat format (default: false)
        :arg master_timeout: Specify timeout for connection to master
        :arg order: The order for this template when merging multiple matching
            ones (higher numbers are merged later, overriding the lower numbers)
        :arg timeout: Explicit operation timeout
        """
        for param in (name, body):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('PUT', _make_path('_template',
            name), params=params, body=body)
        return data

    @query_params('local', 'master_timeout')
    def exists_template(self, name, params=None):
        """
        Return a boolean indicating whether given template exists.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates.html>`_

        :arg name: The name of the template
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        :arg master_timeout: Explicit operation timeout for connection to master
            node
        """
        if name in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'name'.")
        try:
            self.transport.perform_request('HEAD', _make_path('_template',
                name), params=params)
        except NotFoundError:
            return False
        return True

    @query_params('flat_settings', 'local', 'master_timeout')
    def get_template(self, name=None, params=None):
        """
        Retrieve an index template by its name.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates.html>`_

        :arg name: The name of the template
        :arg flat_settings: Return settings in flat format (default: false)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        :arg master_timeout: Explicit operation timeout for connection to master
            node
        """
        _, data = self.transport.perform_request('GET', _make_path('_template',
            name), params=params)
        return data

    @query_params('master_timeout', 'timeout')
    def delete_template(self, name, params=None):
        """
        Delete an index template by its name.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-templates.html>`_

        :arg name: The name of the template
        :arg master_timeout: Specify timeout for connection to master
        :arg timeout: Explicit operation timeout
        """
        if name in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'name'.")
        _, data = self.transport.perform_request('DELETE',
            _make_path('_template', name), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'flat_settings',
        'human', 'ignore_unavailable', 'local')
    def get_settings(self, index=None, name=None, params=None):
        """
        Retrieve settings for one or more (or all) indices.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-get-settings.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg name: The name of the settings that should be included
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default ['open', 'closed'],
            valid choices are: 'open', 'closed', 'none', 'all'
        :arg flat_settings: Return settings in flat format (default: false)
        :arg human: Whether to return version and creation date values in human-
            readable format., default False
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_settings', name), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'flat_settings',
        'ignore_unavailable', 'master_timeout')
    def put_settings(self, body, index=None, params=None):
        """
        Change specific index level settings in real time.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-update-settings.html>`_

        :arg body: The index settings to be updated
        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg flat_settings: Return settings in flat format (default: false)
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg master_timeout: Specify timeout for connection to master
        """
        if body in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'body'.")
        _, data = self.transport.perform_request('PUT', _make_path(index,
            '_settings'), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'master_timeout', 'request_cache')
    def put_warmer(self, name, body, index=None, doc_type=None, params=None):
        """
        Create an index warmer to run registered search requests to warm up the
        index before it is available for search.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-warmers.html>`_

        :arg name: The name of the warmer
        :arg body: The search request definition for the warmer (query, filters,
            facets, sorting, etc)
        :arg index: A comma-separated list of index names to register the warmer
            for; use `_all` or omit to perform the operation on all indices
        :arg doc_type: A comma-separated list of document types to register the
            warmer for; leave empty to perform the operation on all types
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices in the search request
            to warm. (This includes `_all` string or when no indices have been
            specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both, in the search request to
            warm., default 'open', valid choices are: 'open', 'closed', 'none',
            'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed) in the search request
            to warm
        :arg master_timeout: Specify timeout for connection to master
        :arg request_cache: Specify whether the request to be wamred shoyd use
            the request cache, defaults to index level setting
        """
        for param in (name, body):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('PUT', _make_path(index,
            doc_type, '_warmer', name), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'local')
    def get_warmer(self, index=None, doc_type=None, name=None, params=None):
        """
        Retreieve an index warmer.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-warmers.html>`_

        :arg index: A comma-separated list of index names to restrict the
            operation; use `_all` to perform the operation on all indices
        :arg doc_type: A comma-separated list of document types to restrict the
            operation; leave empty to perform the operation on all types
        :arg name: The name of the warmer (supports wildcards); leave empty to
            get all warmers
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            doc_type, '_warmer', name), params=params)
        return data

    @query_params('master_timeout')
    def delete_warmer(self, index, name, params=None):
        """
        Delete an index warmer.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-warmers.html>`_

        :arg index: A comma-separated list of index names to delete warmers from
            (supports wildcards); use `_all` to perform the operation on all
            indices.
        :arg name: A comma-separated list of warmer names to delete (supports
            wildcards); use `_all` to delete all warmers in the specified
            indices. You must specify a name either in the uri or in the
            parameters.
        :arg master_timeout: Specify timeout for connection to master
        """
        for param in (index, name):
            if param in SKIP_IN_PATH:
                raise ValueError("Empty value passed for a required argument.")
        _, data = self.transport.perform_request('DELETE', _make_path(index,
            '_warmer', name), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_indices',
        'ignore_unavailable', 'operation_threading', 'recovery', 'snapshot', 'human')
    def status(self, index=None, params=None):
        """
        Get a comprehensive status information of one or more indices.
        `<http://elastic.co/guide/reference/api/admin-indices-_/>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all` string or
            when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete indices
            that are open, closed or both.
        :arg ignore_indices: When performed on multiple indices, allows to
            ignore `missing` ones, default u'none'
        :arg ignore_unavailable: Whether specified concrete indices should be ignored
            when unavailable (missing or closed)
        :arg operation_threading: TODO: ?
        :arg recovery: Return information about shard recovery
        :arg snapshot: TODO: ?
        :arg human: Whether to return time and byte values in human-readable format.
        """
        _, data = self.transport.perform_request('GET', _make_path(index, '_status'),
            params=params)
        return data

    @query_params('completion_fields', 'fielddata_fields', 'fields', 'groups',
        'human', 'level', 'types')
    def stats(self, index=None, metric=None, params=None):
        """
        Retrieve statistics on different operations happening on an index.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-stats.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg metric: Limit the information returned the specific metrics.
        :arg completion_fields: A comma-separated list of fields for `fielddata`
            and `suggest` index metric (supports wildcards)
        :arg fielddata_fields: A comma-separated list of fields for `fielddata`
            index metric (supports wildcards)
        :arg fields: A comma-separated list of fields for `fielddata` and
            `completion` index metric (supports wildcards)
        :arg groups: A comma-separated list of search groups for `search` index
            metric
        :arg human: Whether to return time and byte values in human-readable
            format., default False
        :arg level: Return stats aggregated at cluster, index or shard level,
            default 'indices', valid choices are: 'cluster', 'indices', 'shards'
        :arg types: A comma-separated list of document types for the `indexing`
            index metric
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_stats', metric), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'human',
        'ignore_unavailable', 'operation_threading')
    def segments(self, index=None, params=None):
        """
        Provide low level segments information that a Lucene index (shard level) is built with.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-segments.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg human: Whether to return time and byte values in human-readable
            format., default False
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg operation_threading: TODO: ?
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_segments'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'flush', 'force',
        'ignore_unavailable', 'max_num_segments', 'only_expunge_deletes',
        'operation_threading', 'wait_for_merge')
    def optimize(self, index=None, params=None):
        """
        Explicitly optimize one or more indices through an API.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-optimize.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg flush: Specify whether the index should be flushed after performing
            the operation (default: true)
        :arg force: Force a merge operation to run, even if there is a single
            segment in the index (default: false)
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg max_num_segments: The number of segments the index should be merged
            into (default: dynamic)
        :arg only_expunge_deletes: Specify whether the operation should only
            expunge deleted documents
        :arg operation_threading: TODO: ?
        :arg wait_for_merge: Specify whether the request should block until the
            merge process is finished (default: true)
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_optimize'), params=params)
        return data

    @query_params('allow_no_indices', 'analyze_wildcard', 'analyzer',
        'default_operator', 'df', 'expand_wildcards', 'explain',
        'ignore_unavailable', 'lenient', 'lowercase_expanded_terms',
        'operation_threading', 'q', 'rewrite')
    def validate_query(self, index=None, doc_type=None, body=None, params=None):
        """
        Validate a potentially expensive query without executing it.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/search-validate.html>`_

        :arg index: A comma-separated list of index names to restrict the
            operation; use `_all` or empty string to perform the operation on
            all indices
        :arg doc_type: A comma-separated list of document types to restrict the
            operation; leave empty to perform the operation on all types
        :arg body: The query definition specified with the Query DSL
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg analyze_wildcard: Specify whether wildcard and prefix queries
            should be analyzed (default: false)
        :arg analyzer: The analyzer to use for the query string
        :arg default_operator: The default operator for query string query (AND
            or OR), default 'OR', valid choices are: 'AND', 'OR'
        :arg df: The field to use as default where no field prefix is given in
            the query string
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg explain: Return detailed information about the error
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg lenient: Specify whether format-based query failures (such as
            providing text to a numeric field) should be ignored
        :arg lowercase_expanded_terms: Specify whether query terms should be
            lowercased
        :arg operation_threading: TODO: ?
        :arg q: Query in the Lucene query string syntax
        :arg rewrite: Provide a more detailed explanation showing the actual
            Lucene query that will be executed.
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            doc_type, '_validate', 'query'), params=params, body=body)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'field_data',
        'fielddata', 'fields', 'filter', 'filter_cache', 'filter_keys', 'id',
        'id_cache', 'ignore_unavailable', 'query_cache', 'recycler')
    def clear_cache(self, index=None, params=None):
        """
        Clear either all caches or specific cached associated with one ore more indices.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-clearcache.html>`_

        :arg index: A comma-separated list of index name to limit the operation
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg field_data: Clear field data
        :arg fielddata: Clear field data
        :arg fields: A comma-separated list of fields to clear when using the
            `field_data` parameter (default: all)
        :arg filter: Clear filter caches
        :arg filter_cache: Clear filter caches
        :arg filter_keys: A comma-separated list of keys to clear when using the
            `filter_cache` parameter (default: all)
        :arg id: Clear ID caches for parent/child
        :arg id_cache: Clear ID caches for parent/child
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg query: Clear query caches
        :arg recycler: Clear the recycler cache
        :arg request: Clear request cache
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_cache', 'clear'), params=params)
        return data

    @query_params('active_only', 'detailed', 'human')
    def recovery(self, index=None, params=None):
        """
        The indices recovery API provides insight into on-going shard
        recoveries. Recovery status may be reported for specific indices, or
        cluster-wide.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-recovery.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg active_only: Display only those recoveries that are currently on-
            going, default False
        :arg detailed: Whether to display detailed information about shard
            recovery, default False
        :arg human: Whether to return time and byte values in human-readable
            format., default False
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_recovery'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'only_ancient_segments', 'wait_for_completion')
    def upgrade(self, index=None, params=None):
        """
        Upgrade one or more indices to the latest format through an API.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-upgrade.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg only_ancient_segments: If true, only ancient (an older Lucene major
            release) segments will be upgraded
        :arg wait_for_completion: Specify whether the request should block until
            the all segments are upgraded (default: false)
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_upgrade'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'human',
        'ignore_unavailable')
    def get_upgrade(self, index=None, params=None):
        """
        Monitor how much of one or more index is upgraded.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-upgrade.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg human: Whether to return time and byte values in human-readable
            format., default False
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_upgrade'), params=params)
        return data

    @query_params()
    def flush_synced(self, index=None, params=None):
        """
        Perform a normal flush, then add a generated unique marker (sync_id) to all shards.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-synced-flush.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string for all indices
        """
        _, data = self.transport.perform_request('POST', _make_path(index,
            '_flush', 'synced'), params=params)
        return data

    @query_params('allow_no_indices', 'expand_wildcards', 'ignore_unavailable',
        'operation_threading', 'status')
    def shard_stores(self, index=None, params=None):
        """
        `<http://www.elastic.co/guide/en/elasticsearch/reference/current/indices-shard-stores.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        :arg operation_threading: TODO: ?
        :arg status: A comma-separated list of statuses used to filter on shards
            to get store information for, valid choices are: 'green', 'yellow',
            'red', 'all'
        """
        _, data = self.transport.perform_request('GET', _make_path(index,
            '_shard_stores'), params=params)
        return data


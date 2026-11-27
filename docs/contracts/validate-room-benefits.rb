#!/usr/bin/env ruby
require 'json'
require 'yaml'

root = File.expand_path(__dir__)
spec = YAML.load_file(File.join(root, 'openapi.yaml'))
examples = JSON.parse(File.read(File.join(root, 'room-benefits-examples.json')))
abort 'contract version mismatch' unless spec.dig('info', 'version') == examples['contract_version']

resolve = lambda do |ref|
  abort "external reference unsupported: #{ref}" unless ref.start_with?('#/')
  result = ref.delete_prefix('#/').split('/').reduce(spec) { |node, key| node.is_a?(Hash) ? node[key] : nil }
  abort "missing reference: #{ref}" unless result
  result
end

walk = lambda do |node|
  case node
  when Hash
    node.each { |key, value| resolve.call(value) if key == '$ref'; walk.call(value) }
  when Array
    node.each { |value| walk.call(value) }
  end
end
walk.call(spec)

spec.fetch('paths').each do |path, item|
  item.each do |method, operation|
    next unless %w[get post put patch delete].include?(method)
    abort "missing 200 response: #{method} #{path}" unless operation.fetch('responses', {}).key?('200')
    if operation['x-implementation-status'] == 'planned' && %w[post put patch delete].include?(method)
      %w[400 401 403 409].each do |code|
        abort "missing #{code} response: #{method} #{path}" unless operation.fetch('responses').key?(code)
      end
    end
    path.scan(/\{([^}]+)\}/).flatten.each do |name|
      parameters = operation.fetch('parameters', []).map { |entry| entry['$ref'] ? resolve.call(entry['$ref']) : entry }
      abort "missing path parameter #{name}: #{method} #{path}" unless parameters.any? { |parameter| parameter['in'] == 'path' && parameter['name'] == name && parameter['required'] }
    end
  end
end

validate = lambda do |schema, value, location|
  schema = resolve.call(schema['$ref']) if schema['$ref']
  next if value.nil? && schema['nullable']
  if schema['allOf']
    schema['allOf'].each { |part| validate.call(part, value, location) }
    next
  end
  if schema['oneOf']
    matches = schema['oneOf'].count do |part|
      begin
        validate.call(part, value, location)
        true
      rescue RuntimeError
        false
      end
    end
    raise "#{location}: expected one matching schema, got #{matches}" unless matches == 1
    next
  end
  raise "#{location}: invalid enum #{value.inspect}" if schema['enum'] && !schema['enum'].include?(value)
  case schema['type']
  when 'object'
    raise "#{location}: expected object" unless value.is_a?(Hash)
    schema.fetch('required', []).each { |field| raise "#{location}: missing #{field}" unless value.key?(field) }
    schema.fetch('properties', {}).each { |field, part| validate.call(part, value[field], "#{location}.#{field}") if value.key?(field) }
  when 'array'
    raise "#{location}: expected array" unless value.is_a?(Array)
    raise "#{location}: too few items" if schema['minItems'] && value.size < schema['minItems']
    raise "#{location}: too many items" if schema['maxItems'] && value.size > schema['maxItems']
    value.each_with_index { |entry, index| validate.call(schema['items'], entry, "#{location}[#{index}]") }
  when 'integer'
    raise "#{location}: expected integer" unless value.is_a?(Integer)
  when 'string'
    raise "#{location}: expected string" unless value.is_a?(String)
  when 'boolean'
    raise "#{location}: expected boolean" unless value == true || value == false
  end
end

cases = examples.fetch('cases')
schemas = spec.fetch('components').fetch('schemas')
{
  'capabilities_off' => schemas.fetch('CapabilitiesResponse'),
  'inventory_inherits_default' => schemas.fetch('InventoryResponse'),
  'quote_request' => schemas.fetch('CreateQuoteRequest'),
  'quote_success' => schemas.fetch('QuoteResponse'),
  'submit_request' => schemas.fetch('SubmitConfiguredBooking'),
  'sold_out' => schemas.fetch('Phase3Error'),
  'feature_disabled' => schemas.fetch('Phase3Error'),
  'stale_version' => schemas.fetch('Phase3Error'),
  'entitlement_none' => schemas.fetch('EntitlementResponse'),
  'claim_request' => spec.dig('paths', '/mini/new-member-entitlement/claim', 'post', 'requestBody', 'content', 'application/json', 'schema'),
  'legacy_booking_blocked_in_production' => schemas.fetch('Phase3Error'),
  'legacy_join_blocked_in_production' => schemas.fetch('Phase3Error')
}.each do |name, schema|
  sample = cases.fetch(name)
  if name.start_with?('legacy_')
    raise "#{name}: expected HTTP 409" unless sample['http_status'] == 409
    sample = sample.fetch('response')
  end
  validate.call(schema, sample, name)
end

puts "#{spec.dig('info', 'version')}: #{spec.fetch('paths').size} paths, references and #{cases.size} examples valid"

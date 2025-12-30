# Log Guardian

The Log Guardian is a powerful log management system designed to provide real-time log monitoring and analysis for your applications. It offers a range of features, including log ingestion, pipeline processing, context enrichment, AI-powered analysis, and notification delivery.

## Project Structure

The Log Guardian is organized into several packages:

- `ingestion`: Contains modules for reading logs from various sources, such as `stdin`, log files, and Unix sockets.
- `pipeline`: Contains modules for processing logs, including log filtering, deduplication, and severity-based prioritization.
- `context`: Contains modules for enriching log context with metadata, such as pod labels and annotations, and source code information.
- `ai`: Contains modules for interacting with AI providers, such as OpenAI, Gemini, and Ollama.
- `notify`: Contains modules for delivering notifications to various channels, such as Slack, Discord, and Hangouts.

## Features

The Log Guardian offers the following key features:

- Log ingestion from various sources, including `stdin`, log files, and Unix sockets.
- Log pipeline processing, including log filtering, deduplication, and severity-based prioritization.
- Context enrichment with metadata, such as pod labels and annotations, and source code information.
- AI-powered analysis using OpenAI, Gemini, and Ollama.
- Notification delivery to various channels, such as Slack, Discord, and Hangouts.

## Documentation

For more detailed documentation on how to use and configure the Log Guardian, refer to the [documentation](/docs).

## Contributing

Contributions to the Log Guardian are welcome. Please see the [contributing guidelines](CONTRIBUTING.md) for more information.

## License

The Log Guardian is licensed under the [MIT License](LICENSE).
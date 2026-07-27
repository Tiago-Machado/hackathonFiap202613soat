# Registro de Decisões de Arquitetura (ADR)

Este documento registra as principais decisões de arquitetura do sistema, o
contexto que as motivou, as alternativas consideradas e as consequências
assumidas. O objetivo é tornar explícito **por que** o sistema é como é —
não apenas o que ele faz.

Formato: cada decisão segue Contexto → Decisão → Alternativas → Consequências.

---

## ADR-001 — Separação em microsserviços desacoplados (API e Worker)

**Contexto.** O protótipo original processava o vídeo de forma síncrona dentro
da requisição HTTP: o usuário enviava o arquivo e ficava aguardando o FFmpeg
terminar. Isso acopla o tempo de resposta ao tempo de processamento, não
escala e derruba a experiência sob carga.

**Decisão.** Dividir o sistema em dois processos independentes: uma **API**,
responsável por receber requisições, autenticar e registrar trabalho; e um
**Worker**, responsável pelo processamento pesado (FFmpeg). Os dois se
comunicam por uma fila de mensagens, sem se conhecerem diretamente.

**Alternativas.** Manter o monolito síncrono (rejeitado: não atende
escalabilidade nem resiliência a picos); usar goroutines dentro do mesmo
processo (rejeitado: o processamento continuaria competindo por recursos com
a API e não escalaria de forma independente).

**Consequências.** API e Worker escalam de forma independente conforme o
gargalo. Em troca, ganha-se a complexidade de operar uma fila e de raciocinar
sobre consistência eventual entre os dois lados.

---

## ADR-002 — Processamento assíncrono com resposta imediata

**Contexto.** Um requisito central é processar vários vídeos ao mesmo tempo e
não perder requisições em pico. Uma resposta síncrona impede ambos.

**Decisão.** O `POST /videos` armazena o vídeo, persiste o job com status
`PENDING`, publica um evento na fila e **retorna imediatamente** o identificador
do job (HTTP 202). O cliente acompanha o resultado consultando `GET /videos`.

**Alternativas.** Resposta síncrona bloqueante (rejeitada pelo requisito);
long-polling ou WebSocket para empurrar o resultado (rejeitado: complexidade
desnecessária para o escopo; a consulta por status resolve).

**Consequências.** A API responde em milissegundos e absorve picos sem
bloquear. O cliente precisa consultar o status — trade-off aceitável e comum
em processamento de mídia.

---

## ADR-003 — RabbitMQ como mensageria (em vez de Kafka)

**Contexto.** É preciso uma fila entre API e Worker. As duas opções naturais
eram RabbitMQ e Apache Kafka.

**Decisão.** Usar **RabbitMQ**.

**Alternativas.** Kafka é excelente para *streaming* de eventos de alto volume
e reprocessamento por offset, mas seu modelo é de log particionado, não de
fila de tarefas. Para o nosso caso — distribuir unidades de trabalho entre
workers concorrentes, com confirmação por mensagem (ack) e reencaminhamento de
falhas — a **semântica de fila de tarefas** do RabbitMQ é mais direta e o
custo operacional é menor. Kafka seria over-engineering aqui.

**Consequências.** Distribuição de trabalho e ack por mensagem simples e
naturais. Caso o produto evolua para *event sourcing* ou altíssima vazão de
eventos, Kafka voltaria à mesa — mas essa não é a necessidade atual.

---

## ADR-004 — Não perder requisição: fila durável, mensagens persistentes e DLQ

**Contexto.** O requisito "em caso de picos, o sistema não deve perder uma
requisição" exige garantias de entrega, inclusive diante de reinício de
serviços ou mensagens problemáticas.

**Decisão.** A fila é **durável**, as mensagens são **persistentes** e o
consumo usa **ack manual** com *prefetch* controlado. Mensagens que falham
repetidamente são desviadas para uma **dead-letter queue** (DLQ) em vez de
serem descartadas ou reprocessadas infinitamente.

**Alternativas.** Ack automático (rejeitado: perderia mensagens se o worker
caísse no meio do processamento); requeue infinito em falha (rejeitado:
mensagem "venenosa" travaria o worker para sempre).

**Consequências.** Uma mensagem só sai da fila quando o trabalho é confirmado;
reinícios não perdem trabalho; falhas persistentes ficam isoladas na DLQ para
inspeção. Em troca, é preciso garantir idempotência e tratar a DLQ
operacionalmente.

---

## ADR-005 — PostgreSQL como fonte de verdade do estado

**Contexto.** É preciso persistir o estado dos jobs, usuários, cotas e
auditoria, com consultas relacionais (listagem por usuário, contagem por dia).

**Decisão.** Usar **PostgreSQL**, com o ciclo de vida do vídeo modelado como
uma **máquina de estados** explícita (`PENDING → PROCESSING → DONE | ERROR`),
cujas transições são validadas no domínio.

**Alternativas.** Um banco de documentos (rejeitado: os dados são relacionais
e as garantias transacionais são desejáveis); guardar estado apenas no storage
de objetos (rejeitado: inviabiliza consultas e integridade).

**Consequências.** Integridade transacional, consultas ricas e um estado
auditável. A máquina de estados no domínio impede transições inválidas
independentemente de quem chama.

---

## ADR-006 — MinIO/S3 para objetos, com dois clientes (interno e público)

**Contexto.** Vídeos e zips não pertencem ao banco relacional. Além disso, o
navegador do usuário precisa baixar o zip diretamente, sem passar pela API.

**Decisão.** Armazenar objetos em **MinIO** (compatível com S3). Usar **dois
clientes**: um com o endpoint **interno** (`minio:9000`) para operações
servidor-a-servidor, e um com o endpoint **público** exclusivamente para gerar
**presigned URLs** que funcionem no navegador do usuário.

**Alternativas.** Servir o download pela própria API (rejeitado: transformaria
a API em gargalo de banda); um único cliente MinIO (rejeitado: a URL assinada
apontaria para um host interno inacessível pelo navegador — foi exatamente o
bug que a separação de clientes resolve).

**Consequências.** Downloads escalam direto pelo storage, sem passar pela
aplicação. O acesso é temporário e assinado. Em produção, o endpoint público
é o domínio real do storage com TLS.

---

## ADR-007 — Redis para rate limiting com política fail-closed

**Contexto.** As rotas autenticadas precisam de proteção contra abuso, com
contadores compartilhados entre múltiplas réplicas da API.

**Decisão.** Rate limiting por usuário usando **Redis** como contador
distribuído. Quando o Redis está indisponível, a política é **fail-closed**:
a requisição é negada.

**Alternativas.** Contador em memória por réplica (rejeitado: não é
consistente entre réplicas); fail-open em falha do Redis (rejeitado: abriria
o sistema justamente quando a proteção é necessária).

**Consequências.** Limite consistente entre réplicas e postura de segurança
conservadora. O trade-off do fail-closed é priorizar segurança sobre
disponibilidade — decisão consciente para um limitador de abuso.

---

## ADR-008 — Clean Architecture (domínio → casos de uso → adapters)

**Contexto.** O sistema precisa ser testável, sustentável e permitir trocar
tecnologias de infraestrutura sem reescrever regra de negócio.

**Decisão.** Organizar o código em camadas: **domínio** (entidades e regras,
sem dependências externas), **casos de uso** (orquestração, dependendo apenas
de **portas**/interfaces) e **adapters** (implementações concretas de Postgres,
RabbitMQ, MinIO, SMTP, JWT). As dependências apontam sempre para dentro.

**Alternativas.** Handlers HTTP chamando o banco diretamente (rejeitado:
acopla regra de negócio a framework e driver, e inviabiliza teste unitário
sem infraestrutura).

**Consequências.** A regra de negócio é testável sem subir nada e independe de
tecnologia. Trocar RabbitMQ por outra fila, por exemplo, afeta apenas um
adapter. O custo é mais estrutura e indireção — que se paga na testabilidade
e na longevidade.

---

## ADR-009 — Autenticação com JWT e senhas em bcrypt

**Contexto.** O sistema deve ser protegido por usuário e senha, e a API é
stateless e replicável.

**Decisão.** Senhas armazenadas com **bcrypt**. Autenticação via **JWT**
assinado (HS256), validado por um middleware antes das rotas protegidas.

**Alternativas.** Sessão em servidor (rejeitado: exigiria estado
compartilhado entre réplicas, contrariando a statelessness); guardar senha com
hash rápido como SHA-256 (rejeitado: inadequado para senhas — bcrypt é
propositalmente lento e salgado).

**Consequências.** Autenticação sem estado no servidor, o que combina com
réplicas horizontais. O token carrega a identidade; a expiração limita a
janela de risco.

---

## ADR-010 — Docker Compose para desenvolvimento, Kubernetes para produção

**Contexto.** É preciso um ambiente reproduzível para desenvolver e demonstrar,
e uma estratégia de execução em produção com escala.

**Decisão.** **Docker Compose** é a base de desenvolvimento e da demonstração
(sobe todo o ecossistema em uma máquina, com um comando). **Kubernetes** é o
alvo de produção, mantido como conjunto de manifests versionados, incluindo um
**HorizontalPodAutoscaler** no Worker.

**Alternativas.** Só Compose (atende o requisito, mas não demonstra a
estratégia de escala em produção); desenvolver diretamente sobre Kubernetes
(rejeitado: adiciona fricção desnecessária ao desenvolvimento local).

**Consequências.** A mesma imagem Docker roda nos dois ambientes, porque a
aplicação é *stateless*, configurada por variáveis de ambiente e escalável por
processo. A escolha entre Compose e Kubernetes é de **topologia de deploy**, não
de arquitetura — o design da aplicação é o mesmo. Essa indiferença ao
orquestrador é uma consequência do design, não um acidente.

---

## ADR-011 — Injeção de relógio e portas como interfaces (testabilidade)

**Contexto.** Regras que dependem de tempo (expiração, timestamps) e de
infraestrutura são difíceis de testar de forma determinística.

**Decisão.** O tempo é injetado como um `Clock` (função que retorna o instante
atual) e todas as dependências dos casos de uso são **interfaces**. Nos testes,
usa-se um relógio fixo e implementações em memória das portas.

**Alternativas.** Chamar `time.Now()` diretamente e instanciar drivers reais
(rejeitado: testes não-determinísticos e dependentes de infraestrutura).

**Consequências.** Os casos de uso e o domínio têm cobertura alta com testes
rápidos, sem subir banco, fila ou storage. O custo é passar dependências
explicitamente — que também melhora a clareza.

---

## ADR-012 — Outbox transacional adiado conscientemente

**Contexto.** No upload, persistir o job no banco e publicar na fila são duas
operações em sistemas distintos. Em teoria, é possível persistir e falhar ao
publicar, gerando um job "órfão".

**Decisão.** Para o escopo atual, **não** implementar o padrão *outbox
transacional*. A decisão é registrada como um próximo passo consciente de
robustez, não como um esquecimento — o esquema de banco já inclui a tabela
`outbox` preparada para essa evolução.

**Alternativas.** Implementar outbox agora (adiaria a entrega sem ganho
proporcional para o cenário de demonstração).

**Consequências.** Simplicidade imediata, com o caminho de robustez mapeado e
o banco já preparado. Um arquiteto reconhece a fronteira do que está pronto e
a documenta, em vez de fingir completude.

---

## ADR-013 — Retenção e auditoria (LGPD)

**Contexto.** O sistema guarda dados de usuários e artefatos de mídia, o que
traz responsabilidades de ciclo de vida e rastreabilidade.

**Decisão.** Objetos e registros têm prazo de expiração (`expires_at`), com um
processo de **purga** periódica no Worker e regra de expiração equivalente no
storage. Eventos sensíveis são registrados em uma tabela de **auditoria**.

**Alternativas.** Reter tudo indefinidamente (rejeitado: risco de privacidade
e custo de armazenamento crescente).

**Consequências.** Ciclo de vida de dados controlado e rastreável. A janela de
retenção é configurável por ambiente.

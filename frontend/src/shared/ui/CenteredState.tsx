export function CenteredState({ title, text }: { title: string; text: string }) {
  return (
    <section className="centered-state">
      <h2>{title}</h2>
      <p>{text}</p>
    </section>
  );
}

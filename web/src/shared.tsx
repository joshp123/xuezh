import type { ReactNode } from "react";

export function Shell(props: { children: ReactNode }) {
  return <main className="app">{props.children}</main>;
}

export function State(props: { title: string; body: string; action?: ReactNode }) {
  return <section className="state"><h2>{props.title}</h2><p>{props.body}</p>{props.action}</section>;
}
